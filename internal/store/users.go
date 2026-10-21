package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"siloam/internal/model"
)

var ErrConflict = errors.New("conflict")

func (s *Store) ensureAuthIndexes(ctx context.Context) error {
	if _, err := s.users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "webauthnId", Value: 1}}, Options: options.Index().SetUnique(true)},
	}); err != nil {
		return err
	}
	_, err := s.sessions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		{Keys: bson.D{{Key: "userId", Value: 1}}},
	})
	return err
}

// ---- users ----

func (s *Store) CountUsers(ctx context.Context, filter bson.D) (int64, error) {
	if filter == nil {
		filter = bson.D{}
	}
	return s.users.CountDocuments(ctx, filter)
}

func (s *Store) CountAdmins(ctx context.Context) (int64, error) {
	return s.CountUsers(ctx, bson.D{{Key: "role", Value: model.RoleAdmin}})
}

func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	cur, err := s.users.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "username", Value: 1}}))
	if err != nil {
		return nil, err
	}
	out := []model.User{}
	return out, cur.All(ctx, &out)
}

func (s *Store) findUser(ctx context.Context, filter bson.D) (*model.User, error) {
	var u model.User
	err := s.users.FindOne(ctx, filter).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUser(ctx context.Context, id bson.ObjectID) (*model.User, error) {
	return s.findUser(ctx, bson.D{{Key: "_id", Value: id}})
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return s.findUser(ctx, bson.D{{Key: "username", Value: username}})
}

func (s *Store) GetUserByWebAuthnID(ctx context.Context, handle []byte) (*model.User, error) {
	return s.findUser(ctx, bson.D{{Key: "webauthnId", Value: handle}})
}

func (s *Store) CreateUser(ctx context.Context, u *model.User) error {
	now := time.Now().UTC()
	u.ID = bson.NewObjectID()
	u.CreatedAt, u.UpdatedAt = now, now
	if u.Passkeys == nil {
		u.Passkeys = []model.Passkey{}
	}
	_, err := s.users.InsertOne(ctx, u)
	if mongo.IsDuplicateKeyError(err) {
		return ErrConflict
	}
	return err
}

// Every user write below is a targeted, atomic update of only the fields it
// owns. Never write back a whole user document read earlier in a request: a
// lock, password reset or passkey change made meanwhile would be undone.

var ErrLastAdmin = errors.New("last admin")

// modifyUser applies update to the user matching filter and returns the
// updated document.
func (s *Store) modifyUser(ctx context.Context, filter bson.D, update any) (*model.User, error) {
	var u model.User
	err := s.users.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&u)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return nil, ErrNotFound
	case mongo.IsDuplicateKeyError(err):
		return nil, ErrConflict
	case err != nil:
		return nil, err
	}
	return &u, nil
}

func set(fields bson.D) bson.D {
	return bson.D{{Key: "$set", Value: append(fields, bson.E{Key: "updatedAt", Value: time.Now().UTC()})}}
}

func byID(id bson.ObjectID) bson.D { return bson.D{{Key: "_id", Value: id}} }

// RecordLogin marks a successful sign-in at now: clears the failed-login
// counter and any expired lock, and sets lastLoginAt — only if no lock is in
// force. A lock that landed while the password was being checked wins:
// ErrLocked is returned.
func (s *Store) RecordLogin(ctx context.Context, id bson.ObjectID, now time.Time) (*model.User, error) {
	u, err := s.modifyUser(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "$or", Value: bson.A{
			bson.D{{Key: "lockedUntil", Value: nil}},
			bson.D{{Key: "lockedUntil", Value: bson.D{{Key: "$lte", Value: now}}}},
		}}},
		set(bson.D{
			{Key: "failedLogins", Value: 0},
			{Key: "lockedAt", Value: nil},
			{Key: "lockedUntil", Value: nil},
			{Key: "lastLoginAt", Value: now},
		}))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrLocked
	}
	return u, err
}

var ErrLocked = errors.New("account locked")

// SetCredentials stores a user's own new username and password.
func (s *Store) SetCredentials(ctx context.Context, id bson.ObjectID, username, hash string) (*model.User, error) {
	return s.modifyUser(ctx, byID(id), set(bson.D{
		{Key: "username", Value: username},
		{Key: "passwordHash", Value: hash},
		{Key: "mustChangePassword", Value: false},
	}))
}

// ResetPassword sets an admin-chosen temporary password and unlocks the
// account; optionally removes all passkeys.
func (s *Store) ResetPassword(ctx context.Context, id bson.ObjectID, hash string, clearPasskeys bool) (*model.User, error) {
	fields := bson.D{
		{Key: "passwordHash", Value: hash},
		{Key: "mustChangePassword", Value: true},
		{Key: "failedLogins", Value: 0},
		{Key: "lockedAt", Value: nil},
		{Key: "lockedUntil", Value: nil},
	}
	if clearPasskeys {
		fields = append(fields, bson.E{Key: "passkeys", Value: bson.A{}})
	}
	return s.modifyUser(ctx, byID(id), set(fields))
}

func (s *Store) SetUsername(ctx context.Context, id bson.ObjectID, username string) (*model.User, error) {
	return s.modifyUser(ctx, byID(id), set(bson.D{{Key: "username", Value: username}}))
}

// SetRole changes a role. Demoting an admin is undone (ErrLastAdmin) if it
// would leave no admin — checked after the write, so two admins demoting each
// other at the same moment cannot both succeed.
func (s *Store) SetRole(ctx context.Context, id bson.ObjectID, role string, kolom int) (*model.User, error) {
	update := set(bson.D{{Key: "role", Value: role}, {Key: "kolom", Value: kolom}})
	if role != model.RoleKolom { // only kolom users have a kolom
		update = bson.D{
			{Key: "$set", Value: bson.D{{Key: "role", Value: role}, {Key: "updatedAt", Value: time.Now().UTC()}}},
			{Key: "$unset", Value: bson.D{{Key: "kolom", Value: ""}}},
		}
	}
	u, err := s.modifyUser(ctx, byID(id), update)
	if err != nil || role == model.RoleAdmin {
		return u, err
	}
	if n, err := s.CountAdmins(ctx); err != nil || n > 0 {
		return u, err
	}
	if _, err := s.modifyUser(ctx, byID(id), set(bson.D{{Key: "role", Value: model.RoleAdmin}})); err != nil {
		return nil, err
	}
	return nil, ErrLastAdmin
}

func (s *Store) AddPasskey(ctx context.Context, id bson.ObjectID, pk model.Passkey) (*model.User, error) {
	return s.modifyUser(ctx, byID(id), bson.D{
		{Key: "$push", Value: bson.D{{Key: "passkeys", Value: pk}}},
		{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}},
	})
}

func passkeyFilter(id bson.ObjectID, passkeyID string) bson.D {
	return bson.D{{Key: "_id", Value: id}, {Key: "passkeys.id", Value: passkeyID}}
}

func (s *Store) RenamePasskey(ctx context.Context, id bson.ObjectID, passkeyID, name string) (*model.User, error) {
	return s.modifyUser(ctx, passkeyFilter(id, passkeyID), set(bson.D{{Key: "passkeys.$.name", Value: name}}))
}

// UsePasskey stores the credential's updated sign counter/flags after a sign-in.
func (s *Store) UsePasskey(ctx context.Context, id bson.ObjectID, passkeyID string, credential []byte) error {
	_, err := s.modifyUser(ctx, passkeyFilter(id, passkeyID), set(bson.D{
		{Key: "passkeys.$.credential", Value: credential},
		{Key: "passkeys.$.lastUsedAt", Value: time.Now().UTC()},
	}))
	return err
}

func (s *Store) DeletePasskey(ctx context.Context, id bson.ObjectID, passkeyID string) (*model.User, error) {
	return s.modifyUser(ctx, passkeyFilter(id, passkeyID), bson.D{
		{Key: "$pull", Value: bson.D{{Key: "passkeys", Value: bson.D{{Key: "id", Value: passkeyID}}}}},
		{Key: "$set", Value: bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}},
	})
}

// DeleteUser removes a user and their sessions. Deleting an admin is undone
// (ErrLastAdmin) if it would leave no admin, as in SetRole.
func (s *Store) DeleteUser(ctx context.Context, id bson.ObjectID) error {
	var u model.User
	err := s.users.FindOneAndDelete(ctx, byID(id)).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if u.IsAdmin() {
		n, err := s.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if n == 0 {
			if _, err := s.users.InsertOne(ctx, u); err != nil {
				return err
			}
			return ErrLastAdmin
		}
	}
	return s.DeleteUserSessions(ctx, id, "")
}

// RecordFailedLogin atomically counts a wrong password at now. Once the count
// reaches maxAttempts the account is locked until now+lockFor. A lock that has
// already expired is cleared first, so counting starts again from 1. It
// returns the updated user.
func (s *Store) RecordFailedLogin(ctx context.Context, id bson.ObjectID, maxAttempts int, now time.Time, lockFor time.Duration) (*model.User, error) {
	until := bson.D{{Key: "$ifNull", Value: bson.A{"$lockedUntil", nil}}}
	hasLock := bson.D{{Key: "$ne", Value: bson.A{until, nil}}}
	active := bson.D{{Key: "$and", Value: bson.A{hasLock, bson.D{{Key: "$gt", Value: bson.A{"$lockedUntil", now}}}}}}
	failed := bson.D{{Key: "$ifNull", Value: bson.A{"$failedLogins", 0}}}
	cond := func(ifExpr, thenExpr, elseExpr any) bson.D {
		return bson.D{{Key: "$cond", Value: bson.A{ifExpr, thenExpr, elseExpr}}}
	}
	plusOne := func(v any) bson.D { return bson.D{{Key: "$add", Value: bson.A{v, 1}}} }
	lockNow := bson.D{{Key: "$and", Value: bson.A{
		bson.D{{Key: "$eq", Value: bson.A{"$lockedUntil", nil}}},
		bson.D{{Key: "$gte", Value: bson.A{"$failedLogins", maxAttempts}}},
	}}}
	return s.modifyUser(ctx, byID(id), mongo.Pipeline{
		// 1. count this failure; drop an expired lock and restart the count after it
		{{Key: "$set", Value: bson.D{
			{Key: "failedLogins", Value: cond(active, plusOne(failed), cond(hasLock, 1, plusOne(failed)))},
			{Key: "lockedUntil", Value: cond(active, "$lockedUntil", nil)},
			{Key: "lockedAt", Value: cond(active, "$lockedAt", nil)},
		}}},
		// 2. lock once the limit is reached (an active lock is never extended)
		{{Key: "$set", Value: bson.D{
			{Key: "lockedUntil", Value: cond(lockNow, now.Add(lockFor), "$lockedUntil")},
			{Key: "lockedAt", Value: cond(lockNow, now, "$lockedAt")},
			{Key: "updatedAt", Value: now},
		}}},
	})
}

// ---- sessions ----

func (s *Store) CreateSession(ctx context.Context, sess model.Session) error {
	_, err := s.sessions.InsertOne(ctx, sess)
	return err
}

// GetSession returns an unexpired session (the TTL index removes old ones lazily).
func (s *Store) GetSession(ctx context.Context, id string) (*model.Session, error) {
	var sess model.Session
	err := s.sessions.FindOne(ctx, bson.D{
		{Key: "_id", Value: id},
		{Key: "expiresAt", Value: bson.D{{Key: "$gt", Value: time.Now().UTC()}}},
	}).Decode(&sess)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.sessions.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	return err
}

// DeleteUserSessions signs a user out everywhere, except the session keepID.
func (s *Store) DeleteUserSessions(ctx context.Context, userID bson.ObjectID, keepID string) error {
	filter := bson.D{{Key: "userId", Value: userID}}
	if keepID != "" {
		filter = append(filter, bson.E{Key: "_id", Value: bson.D{{Key: "$ne", Value: keepID}}})
	}
	_, err := s.sessions.DeleteMany(ctx, filter)
	return err
}

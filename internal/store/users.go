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
		{Keys: bson.D{{Key: "calendarToken", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
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

// MigrateUserScopes moves the old one-kolom ("kolom") and one-category
// ("bipra") users to the scoped role with a list of one. Idempotent.
func (s *Store) MigrateUserScopes(ctx context.Context) error {
	var old []struct {
		ID      bson.ObjectID  `bson:"_id"`
		Role    string         `bson:"role"`
		Kolom   int            `bson:"kolom"`
		BipraID *bson.ObjectID `bson:"bipraId"`
	}
	cur, err := s.users.Find(ctx, bson.D{{Key: "role", Value: bson.D{{Key: "$in", Value: bson.A{model.LegacyRoleKolom, model.LegacyRoleBipra}}}}})
	if err != nil {
		return err
	}
	if err := cur.All(ctx, &old); err != nil {
		return err
	}
	for _, u := range old {
		var koloms []int
		var bipras []bson.ObjectID
		if u.Role == model.LegacyRoleKolom && u.Kolom > 0 {
			koloms = []int{u.Kolom}
		}
		if u.Role == model.LegacyRoleBipra && u.BipraID != nil {
			bipras = []bson.ObjectID{*u.BipraID}
		}
		// without either the account could change nothing: a regular user's
		// rights would be more than it had, so it keeps an empty scope
		if _, err := s.SetRole(ctx, u.ID, model.RoleScoped, koloms, bipras); err != nil {
			return err
		}
	}
	return nil
}

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

// RecordLogin marks a successful sign-in at now from source (model.LoginWeb
// or LoginAndroid): clears the failed-login counter and any expired lock,
// sets lastLoginAt, and logs it for the user statistics — only if no lock is
// in force. A lock that landed while the password was being checked wins:
// ErrLocked is returned.
func (s *Store) RecordLogin(ctx context.Context, id bson.ObjectID, now time.Time, source string) (*model.User, error) {
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
			{Key: "lastLoginSource", Value: source},
		}))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrLocked
	}
	if err != nil {
		return nil, err
	}
	_, err = s.logins().InsertOne(ctx, model.Login{
		UserID: u.ID, Username: u.Username, Role: u.Role, Koloms: u.Koloms, BipraIDs: u.BipraIDs, Source: source, At: now,
	})
	return u, err
}

func (s *Store) logins() *mongo.Collection { return s.users.Database().Collection("logins") }

// ensureLoginIndexes indexes the sign-in log by time. The log is kept (no
// expiry): an expiring index from an earlier version is replaced.
func (s *Store) ensureLoginIndexes(ctx context.Context) error {
	cur, err := s.logins().Indexes().List(ctx)
	if err != nil {
		return err
	}
	var idx []bson.M
	if err := cur.All(ctx, &idx); err != nil {
		return err
	}
	for _, ix := range idx {
		if _, ttl := ix["expireAfterSeconds"]; ttl {
			name, _ := ix["name"].(string)
			if err := s.logins().Indexes().DropOne(ctx, name); err != nil {
				return err
			}
		}
	}
	_, err = s.logins().Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "at", Value: -1}}})
	return err
}

// ListLogins returns the sign-ins from from (inclusive) to to (exclusive), newest first.
func (s *Store) ListLogins(ctx context.Context, from, to time.Time) ([]model.Login, error) {
	cur, err := s.logins().Find(ctx,
		bson.D{{Key: "at", Value: bson.D{{Key: "$gte", Value: from}, {Key: "$lt", Value: to}}}},
		options.Find().SetSort(bson.D{{Key: "at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	out := []model.Login{}
	return out, cur.All(ctx, &out)
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

// GetUserByCalendarToken finds the user whose calendar feed link has token.
func (s *Store) GetUserByCalendarToken(ctx context.Context, token string) (*model.User, error) {
	if token == "" {
		return nil, ErrNotFound
	}
	return s.findUser(ctx, bson.D{{Key: "calendarToken", Value: token}})
}

// SetCalendarToken gives the user a new calendar feed link (the old one stops working).
func (s *Store) SetCalendarToken(ctx context.Context, id bson.ObjectID, token string) (*model.User, error) {
	return s.modifyUser(ctx, byID(id), set(bson.D{{Key: "calendarToken", Value: token}}))
}

// EnsureCalendarToken gives the user token as their calendar feed secret
// unless they already have one, in one atomic step (two first requests at
// once cannot leave two different tokens), and returns the user.
func (s *Store) EnsureCalendarToken(ctx context.Context, id bson.ObjectID, token string) (*model.User, error) {
	u, err := s.modifyUser(ctx,
		bson.D{{Key: "_id", Value: id}, {Key: "calendarToken", Value: bson.D{{Key: "$exists", Value: false}}}},
		set(bson.D{{Key: "calendarToken", Value: token}}))
	if errors.Is(err, ErrNotFound) { // already has one (or is gone: GetUser tells)
		return s.GetUser(ctx, id)
	}
	return u, err
}

func (s *Store) SetUsername(ctx context.Context, id bson.ObjectID, username string) (*model.User, error) {
	return s.modifyUser(ctx, byID(id), set(bson.D{{Key: "username", Value: username}}))
}

// SetRole changes a role. Demoting an admin is undone (ErrLastAdmin) if it
// would leave no admin — checked after the write, so two admins demoting each
// other at the same moment cannot both succeed.
func (s *Store) SetRole(ctx context.Context, id bson.ObjectID, role string, koloms []int, bipras []bson.ObjectID) (*model.User, error) {
	// only scoped users and viewers have koloms and categories (the old
	// one-kolom and one-category fields go too)
	fields := bson.D{{Key: "role", Value: role}, {Key: "updatedAt", Value: time.Now().UTC()}}
	unset := bson.D{{Key: "kolom", Value: ""}, {Key: "bipraId", Value: ""}}
	if model.HasScope(role) && len(koloms) > 0 {
		fields = append(fields, bson.E{Key: "koloms", Value: koloms})
	} else {
		unset = append(unset, bson.E{Key: "koloms", Value: ""})
	}
	if model.HasScope(role) && len(bipras) > 0 {
		fields = append(fields, bson.E{Key: "bipraIds", Value: bipras})
	} else {
		unset = append(unset, bson.E{Key: "bipraIds", Value: ""})
	}
	update := bson.D{{Key: "$set", Value: fields}, {Key: "$unset", Value: unset}}
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

// ActiveSession is a user's unexpired sign-in (signed in now).
type ActiveSession struct {
	UserID    bson.ObjectID `bson:"userId"`
	Source    string        `bson:"source"`
	CreatedAt time.Time     `bson:"createdAt"`
}

// ListActiveSessions returns the sessions that have not expired, newest first.
func (s *Store) ListActiveSessions(ctx context.Context, now time.Time) ([]ActiveSession, error) {
	cur, err := s.sessions.Find(ctx, bson.D{{Key: "expiresAt", Value: bson.D{{Key: "$gt", Value: now}}}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	out := []ActiveSession{}
	return out, cur.All(ctx, &out)
}

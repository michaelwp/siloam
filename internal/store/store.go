// Package store persists families and members in MongoDB.
package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"

	"siloam/internal/model"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	client   *mongo.Client
	db       string
	families *mongo.Collection
	members  *mongo.Collection
	users    *mongo.Collection
	sessions *mongo.Collection
}

// DefaultDatabase is used when the URI has no database path.
const DefaultDatabase = "siloam"

// DatabaseName returns the database in the URI path
// (mongodb://host/<name>), or DefaultDatabase.
func DatabaseName(uri string) (string, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return "", err
	}
	if cs.Database == "" {
		return DefaultDatabase, nil
	}
	return cs.Database, nil
}

// Connect opens the database without modifying it. dbName "" means the
// database named in the URI.
func Connect(ctx context.Context, uri, dbName string) (*Store, error) {
	if dbName == "" {
		var err error
		if dbName, err = DatabaseName(uri); err != nil {
			return nil, err
		}
	}
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	db := client.Database(dbName)
	return &Store{
		client:   client,
		db:       dbName,
		families: db.Collection("families"),
		members:  db.Collection("members"),
		users:    db.Collection("users"),
		sessions: db.Collection("sessions"),
	}, nil
}

// Open connects and ensures the indexes the app relies on.
func Open(ctx context.Context, uri, dbName string) (*Store, error) {
	s, err := Connect(ctx, uri, dbName)
	if err != nil {
		return nil, err
	}
	if _, err := s.members.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "familyId", Value: 1}, {Key: "order", Value: 1}},
	}); err != nil {
		_ = s.Close(ctx)
		return nil, err
	}
	if err := s.ensureAuthIndexes(ctx); err != nil {
		_ = s.Close(ctx)
		return nil, err
	}
	return s, nil
}

// Database is the name of the connected database.
func (s *Store) Database() string { return s.db }

func (s *Store) Close(ctx context.Context) error { return s.client.Disconnect(ctx) }

func (s *Store) Ping(ctx context.Context) error { return s.client.Ping(ctx, nil) }

// ---- families ----

func (s *Store) ListFamilies(ctx context.Context) ([]model.Family, error) {
	cur, err := s.families.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "no", Value: 1}}))
	if err != nil {
		return nil, err
	}
	out := []model.Family{}
	return out, cur.All(ctx, &out)
}

func (s *Store) GetFamily(ctx context.Context, id bson.ObjectID) (model.Family, error) {
	var f model.Family
	err := s.families.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&f)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return f, ErrNotFound
	}
	return f, err
}

func (s *Store) CreateFamily(ctx context.Context, f model.Family) (model.Family, error) {
	now := time.Now().UTC()
	f.ID = bson.NewObjectID()
	f.CreatedAt, f.UpdatedAt = now, now
	if f.No <= 0 {
		n, err := s.nextNumber(ctx, s.families, bson.D{}, "no")
		if err != nil {
			return f, err
		}
		f.No = n
	}
	_, err := s.families.InsertOne(ctx, f)
	return f, err
}

func (s *Store) UpdateFamily(ctx context.Context, f model.Family) (model.Family, error) {
	old, err := s.GetFamily(ctx, f.ID)
	if err != nil {
		return f, err
	}
	f.CreatedAt = old.CreatedAt
	f.UpdatedAt = time.Now().UTC()
	if f.No <= 0 {
		f.No = old.No
	}
	_, err = s.families.ReplaceOne(ctx, bson.D{{Key: "_id", Value: f.ID}}, f)
	return f, err
}

// DeleteFamily removes the family and its members, then renumbers the rest.
func (s *Store) DeleteFamily(ctx context.Context, id bson.ObjectID) error {
	res, err := s.families.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	if _, err := s.members.DeleteMany(ctx, bson.D{{Key: "familyId", Value: id}}); err != nil {
		return err
	}
	fams, err := s.ListFamilies(ctx)
	if err != nil {
		return err
	}
	for i, f := range fams {
		if f.No != i+1 {
			if _, err := s.families.UpdateByID(ctx, f.ID, bson.D{{Key: "$set", Value: bson.D{{Key: "no", Value: i + 1}}}}); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---- members ----

// ListMembers returns members sorted by family order; familyID may be zero for all.
func (s *Store) ListMembers(ctx context.Context, familyID bson.ObjectID) ([]model.Member, error) {
	filter := bson.D{}
	if !familyID.IsZero() {
		filter = bson.D{{Key: "familyId", Value: familyID}}
	}
	cur, err := s.members.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "familyId", Value: 1}, {Key: "order", Value: 1}}))
	if err != nil {
		return nil, err
	}
	out := []model.Member{}
	return out, cur.All(ctx, &out)
}

func (s *Store) GetMember(ctx context.Context, id bson.ObjectID) (model.Member, error) {
	var m model.Member
	err := s.members.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&m)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return m, ErrNotFound
	}
	return m, err
}

func (s *Store) CreateMember(ctx context.Context, m model.Member) (model.Member, error) {
	if _, err := s.GetFamily(ctx, m.FamilyID); err != nil {
		return m, err
	}
	now := time.Now().UTC()
	m.ID = bson.NewObjectID()
	m.CreatedAt, m.UpdatedAt = now, now
	if m.Order <= 0 {
		n, err := s.nextNumber(ctx, s.members, bson.D{{Key: "familyId", Value: m.FamilyID}}, "order")
		if err != nil {
			return m, err
		}
		m.Order = n
	}
	_, err := s.members.InsertOne(ctx, m)
	return m, err
}

func (s *Store) UpdateMember(ctx context.Context, m model.Member) (model.Member, error) {
	old, err := s.GetMember(ctx, m.ID)
	if err != nil {
		return m, err
	}
	if _, err := s.GetFamily(ctx, m.FamilyID); err != nil {
		return m, err
	}
	m.CreatedAt = old.CreatedAt
	m.UpdatedAt = time.Now().UTC()
	if m.FamilyID != old.FamilyID {
		// moved to another family: append at the end there
		n, err := s.nextNumber(ctx, s.members, bson.D{{Key: "familyId", Value: m.FamilyID}}, "order")
		if err != nil {
			return m, err
		}
		m.Order = n
	} else if m.Order <= 0 {
		m.Order = old.Order
	}
	if _, err := s.members.ReplaceOne(ctx, bson.D{{Key: "_id", Value: m.ID}}, m); err != nil {
		return m, err
	}
	if m.FamilyID != old.FamilyID {
		return m, s.renumberMembers(ctx, old.FamilyID)
	}
	return m, nil
}

func (s *Store) DeleteMember(ctx context.Context, id bson.ObjectID) error {
	m, err := s.GetMember(ctx, id)
	if err != nil {
		return err
	}
	if _, err := s.members.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}}); err != nil {
		return err
	}
	return s.renumberMembers(ctx, m.FamilyID)
}

func (s *Store) renumberMembers(ctx context.Context, familyID bson.ObjectID) error {
	ms, err := s.ListMembers(ctx, familyID)
	if err != nil {
		return err
	}
	for i, m := range ms {
		if m.Order != i+1 {
			if _, err := s.members.UpdateByID(ctx, m.ID, bson.D{{Key: "$set", Value: bson.D{{Key: "order", Value: i + 1}}}}); err != nil {
				return err
			}
		}
	}
	return nil
}

// nextNumber returns the highest value of field among docs matching filter, plus one.
func (s *Store) nextNumber(ctx context.Context, c *mongo.Collection, filter bson.D, field string) (int, error) {
	var top bson.M
	err := c.FindOne(ctx, filter, options.FindOne().
		SetSort(bson.D{{Key: field, Value: -1}}).
		SetProjection(bson.D{{Key: field, Value: 1}})).Decode(&top)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	switch v := top[field].(type) {
	case int32:
		return int(v) + 1, nil
	case int64:
		return int(v) + 1, nil
	}
	return 1, nil
}

// MemberCounts returns the number of members per family in one query.
func (s *Store) MemberCounts(ctx context.Context) (map[bson.ObjectID]int, error) {
	cur, err := s.members.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$familyId"}, {Key: "n", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
	})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID bson.ObjectID `bson:"_id"`
		N  int           `bson:"n"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[bson.ObjectID]int, len(rows))
	for _, r := range rows {
		out[r.ID] = r.N
	}
	return out, nil
}

func (s *Store) CountMembers(ctx context.Context, familyID bson.ObjectID) (int, error) {
	n, err := s.members.CountDocuments(ctx, bson.D{{Key: "familyId", Value: familyID}})
	return int(n), err
}

// InsertMembers bulk-inserts members of one family as given (used by the importer).
func (s *Store) InsertMembers(ctx context.Context, familyID bson.ObjectID, ms []model.Member) error {
	if len(ms) == 0 {
		return nil
	}
	now := time.Now().UTC()
	docs := make([]any, len(ms))
	for i := range ms {
		ms[i].ID = bson.NewObjectID()
		ms[i].FamilyID = familyID
		ms[i].CreatedAt, ms[i].UpdatedAt = now, now
		if ms[i].Order <= 0 {
			ms[i].Order = i + 1
		}
		docs[i] = ms[i]
	}
	_, err := s.members.InsertMany(ctx, docs)
	return err
}

// ---- bulk (import) ----

func (s *Store) IsEmpty(ctx context.Context) (bool, error) {
	n, err := s.families.CountDocuments(ctx, bson.D{})
	return n == 0, err
}

func (s *Store) Reset(ctx context.Context) error {
	if _, err := s.members.DeleteMany(ctx, bson.D{}); err != nil {
		return err
	}
	_, err := s.families.DeleteMany(ctx, bson.D{})
	return err
}

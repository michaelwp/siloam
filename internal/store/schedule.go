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

var (
	ErrTypeExists = errors.New("worship type already exists")
	ErrTypeInUse  = errors.New("worship type is used by services")
	ErrNoType     = errors.New("worship type does not exist")
)

func (s *Store) ensureScheduleIndexes(ctx context.Context) error {
	if _, err := s.worshipTypes.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "name", Value: 1}},
		// case-insensitive: "pemuda" and "Pemuda" are the same type
		Options: options.Index().SetUnique(true).SetCollation(&options.Collation{Locale: "id", Strength: 2}),
	}); err != nil {
		return err
	}
	_, err := s.services.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "date", Value: 1}, {Key: "time", Value: 1}},
	})
	return err
}

// seededMarker records in the meta collection that the default worship types
// were added once, so types an admin deleted never come back.
const seededMarker = "worshipTypesSeeded"

// SeedWorshipTypes adds the default types the first time a database is used
// (an empty list). It runs when the server starts (not on Open), so a copy can
// bring the source's own types first.
func (s *Store) SeedWorshipTypes(ctx context.Context) error {
	meta := s.worshipTypes.Database().Collection("meta")
	n, err := meta.CountDocuments(ctx, bson.D{{Key: "_id", Value: seededMarker}})
	if err != nil || n > 0 {
		return err
	}
	if n, err = s.worshipTypes.CountDocuments(ctx, bson.D{}); err != nil {
		return err
	}
	if n == 0 {
		for i, t := range model.DefaultWorshipTypes {
			t.Order = i + 1
			if _, err := s.CreateWorshipType(ctx, t); err != nil && !errors.Is(err, ErrTypeExists) {
				return err
			}
		}
	}
	_, err = meta.UpdateOne(ctx, bson.D{{Key: "_id", Value: seededMarker}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "at", Value: time.Now().UTC()}}}},
		options.UpdateOne().SetUpsert(true))
	return err
}

// ---- worship types ----

func (s *Store) ListWorshipTypes(ctx context.Context) ([]model.WorshipType, error) {
	cur, err := s.worshipTypes.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	out := []model.WorshipType{}
	return out, cur.All(ctx, &out)
}

func (s *Store) GetWorshipType(ctx context.Context, id bson.ObjectID) (model.WorshipType, error) {
	var t model.WorshipType
	err := s.worshipTypes.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&t)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return t, ErrNoType
	}
	return t, err
}

func (s *Store) CreateWorshipType(ctx context.Context, t model.WorshipType) (model.WorshipType, error) {
	t.ID = bson.NewObjectID()
	t.CreatedAt = time.Now().UTC()
	if t.Order <= 0 {
		n, err := s.nextNumber(ctx, s.worshipTypes, bson.D{}, "order")
		if err != nil {
			return t, err
		}
		t.Order = n
	}
	_, err := s.worshipTypes.InsertOne(ctx, t)
	if mongo.IsDuplicateKeyError(err) {
		return t, ErrTypeExists
	}
	return t, err
}

// UpdateWorshipType changes the name and color (services refer to the type
// by ID, so they follow the change).
func (s *Store) UpdateWorshipType(ctx context.Context, t model.WorshipType) (model.WorshipType, error) {
	var out model.WorshipType
	err := s.worshipTypes.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: t.ID}},
		typeUpdate(t),
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&out)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return out, ErrNotFound
	case mongo.IsDuplicateKeyError(err):
		return out, ErrTypeExists
	}
	return out, err
}

// typeUpdate sets a type's name, color and BIPRA link (or removes the link).
func typeUpdate(t model.WorshipType) bson.D {
	u := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: t.Name}, {Key: "color", Value: t.Color}}}}
	if t.BipraID != nil {
		u[0].Value = append(u[0].Value.(bson.D), bson.E{Key: "bipraId", Value: *t.BipraID})
	} else {
		u = append(u, bson.E{Key: "$unset", Value: bson.D{{Key: "bipraId", Value: ""}}})
	}
	return u
}

const typesLinkedMarker = "worshipTypesBipraLinked"

// LinkWorshipTypesToBipra links, once, the jenis ibadah without a link to the
// BIPRA category their name names (model.GuessBipra): "Pemuda" to Pemuda,
// "Anak Sekolah Minggu (ASM)" to Anak (ASM), … Admins change links on Jenis
// Ibadah afterwards. Runs at server start, after SeedBipra.
func (s *Store) LinkWorshipTypesToBipra(ctx context.Context) error {
	n, err := s.meta().CountDocuments(ctx, bson.D{{Key: "_id", Value: typesLinkedMarker}})
	if err != nil || n > 0 {
		return err
	}
	rules, err := s.BipraRules(ctx)
	if err != nil {
		return err
	}
	types, err := s.ListWorshipTypes(ctx)
	if err != nil {
		return err
	}
	for _, t := range types {
		if t.BipraID != nil {
			continue
		}
		if id := model.GuessBipra(t.Name, rules); id != nil {
			if _, err := s.worshipTypes.UpdateOne(ctx, bson.D{{Key: "_id", Value: t.ID}},
				bson.D{{Key: "$set", Value: bson.D{{Key: "bipraId", Value: *id}}}}); err != nil {
				return err
			}
		}
	}
	_, err = s.meta().UpdateOne(ctx, bson.D{{Key: "_id", Value: typesLinkedMarker}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "at", Value: time.Now().UTC()}}}},
		options.UpdateOne().SetUpsert(true))
	return err
}

// DeleteWorshipType removes a type no service uses.
func (s *Store) DeleteWorshipType(ctx context.Context, id bson.ObjectID) error {
	n, err := s.services.CountDocuments(ctx, bson.D{{Key: "typeId", Value: id}})
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrTypeInUse
	}
	res, err := s.worshipTypes.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- services ----

func (s *Store) ListServices(ctx context.Context) ([]model.Service, error) {
	cur, err := s.services.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "date", Value: 1}, {Key: "time", Value: 1}}))
	if err != nil {
		return nil, err
	}
	out := []model.Service{}
	return out, cur.All(ctx, &out)
}

func (s *Store) GetService(ctx context.Context, id bson.ObjectID) (model.Service, error) {
	var sv model.Service
	err := s.services.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&sv)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return sv, ErrNotFound
	}
	return sv, err
}

func (s *Store) checkServiceRefs(ctx context.Context, sv model.Service) error {
	if sv.Kolom != 0 { // 0: the whole congregation (the API checks it is allowed)
		if err := s.requireKolom(ctx, sv.Kolom); err != nil {
			return err
		}
	}
	_, err := s.GetWorshipType(ctx, sv.TypeID)
	return err
}

func (s *Store) CreateService(ctx context.Context, sv model.Service) (model.Service, error) {
	if err := s.checkServiceRefs(ctx, sv); err != nil {
		return sv, err
	}
	now := time.Now().UTC()
	sv.ID = bson.NewObjectID()
	sv.CreatedAt, sv.UpdatedAt = now, now
	_, err := s.services.InsertOne(ctx, sv)
	return sv, err
}

func (s *Store) UpdateService(ctx context.Context, sv model.Service) (model.Service, error) {
	old, err := s.GetService(ctx, sv.ID)
	if err != nil {
		return sv, err
	}
	if err := s.checkServiceRefs(ctx, sv); err != nil {
		return sv, err
	}
	sv.CreatedAt = old.CreatedAt
	sv.UpdatedAt = time.Now().UTC()
	_, err = s.services.ReplaceOne(ctx, bson.D{{Key: "_id", Value: sv.ID}}, sv)
	return sv, err
}

func (s *Store) DeleteService(ctx context.Context, id bson.ObjectID) error {
	res, err := s.services.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// CountServices is used by the schedule importer to refuse a second import.
func (s *Store) CountServices(ctx context.Context) (int, error) {
	n, err := s.services.CountDocuments(ctx, bson.D{})
	return int(n), err
}

// ResetServices deletes the whole schedule (types are kept).
func (s *Store) ResetServices(ctx context.Context) error {
	_, err := s.services.DeleteMany(ctx, bson.D{})
	return err
}

// InsertServices bulk-inserts an imported schedule.
func (s *Store) InsertServices(ctx context.Context, svs []model.Service) error {
	if len(svs) == 0 {
		return nil
	}
	now := time.Now().UTC()
	docs := make([]any, len(svs))
	for i := range svs {
		svs[i].ID = bson.NewObjectID()
		svs[i].CreatedAt, svs[i].UpdatedAt = now, now
		docs[i] = svs[i]
	}
	_, err := s.services.InsertMany(ctx, docs)
	return err
}

// MemberNames returns the names of the given members (missing ones are left out).
func (s *Store) MemberNames(ctx context.Context, ids []bson.ObjectID) (map[bson.ObjectID]string, error) {
	cur, err := s.members.Find(ctx,
		bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}},
		options.Find().SetProjection(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ID   bson.ObjectID `bson:"_id"`
		Name string        `bson:"name"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[bson.ObjectID]string, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Name
	}
	return out, nil
}

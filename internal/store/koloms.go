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

const (
	// DefaultKoloms is how many koloms (1..n) a new database starts with.
	DefaultKoloms = 12
	// LegacyKolom owns the families created before koloms existed.
	LegacyKolom = 11
)

var (
	ErrKolomExists = errors.New("kolom already exists")
	ErrKolomInUse  = errors.New("kolom still has families")
	ErrNoKolom     = errors.New("kolom does not exist")
)

func (s *Store) ensureKolomIndexes(ctx context.Context) error {
	_, err := s.koloms.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "number", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// MigrateKoloms seeds koloms 1..DefaultKoloms into an empty list, moves
// families without a kolom into LegacyKolom, and adds any kolom a family uses
// that is missing from the list (e.g. after copying data). Safe to repeat.
func (s *Store) MigrateKoloms(ctx context.Context) error {
	n, err := s.koloms.CountDocuments(ctx, bson.D{})
	if err != nil {
		return err
	}
	if n == 0 {
		for i := 1; i <= DefaultKoloms; i++ {
			if err := s.upsertKolom(ctx, i); err != nil {
				return err
			}
		}
	}
	if _, err := s.families.UpdateMany(ctx,
		bson.D{{Key: "kolom", Value: bson.D{{Key: "$in", Value: bson.A{nil, 0}}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "kolom", Value: LegacyKolom}}}},
	); err != nil {
		return err
	}
	var used []int
	if err := s.families.Distinct(ctx, "kolom", bson.D{}).Decode(&used); err != nil {
		return err
	}
	for _, k := range used {
		if err := s.upsertKolom(ctx, k); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) upsertKolom(ctx context.Context, number int) error {
	_, err := s.koloms.UpdateOne(ctx,
		bson.D{{Key: "number", Value: number}},
		bson.D{{Key: "$setOnInsert", Value: bson.D{
			{Key: "_id", Value: bson.NewObjectID()},
			{Key: "createdAt", Value: time.Now().UTC()},
		}}},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *Store) ListKoloms(ctx context.Context) ([]model.Kolom, error) {
	cur, err := s.koloms.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "number", Value: 1}}))
	if err != nil {
		return nil, err
	}
	out := []model.Kolom{}
	return out, cur.All(ctx, &out)
}

func (s *Store) KolomExists(ctx context.Context, number int) (bool, error) {
	n, err := s.koloms.CountDocuments(ctx, bson.D{{Key: "number", Value: number}})
	return n > 0, err
}

func (s *Store) CreateKolom(ctx context.Context, number int) (model.Kolom, error) {
	k := model.Kolom{ID: bson.NewObjectID(), Number: number, CreatedAt: time.Now().UTC()}
	_, err := s.koloms.InsertOne(ctx, k)
	if mongo.IsDuplicateKeyError(err) {
		return k, ErrKolomExists
	}
	return k, err
}

// DeleteKolom removes a kolom that no family belongs to.
func (s *Store) DeleteKolom(ctx context.Context, id bson.ObjectID) error {
	var k model.Kolom
	err := s.koloms.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&k)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	n, err := s.families.CountDocuments(ctx, bson.D{{Key: "kolom", Value: k.Number}})
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrKolomInUse
	}
	_, err = s.koloms.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	return err
}

// CreateKolomIfMissing adds the kolom unless it already exists.
func (s *Store) CreateKolomIfMissing(ctx context.Context, number int) error {
	if number < 1 {
		return errors.New("kolom must be 1 or more")
	}
	return s.upsertKolom(ctx, number)
}

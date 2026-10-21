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

// ErrBipraExists: another category already has this code (case ignored).
var ErrBipraExists = errors.New("bipra category key already exists")

const bipraSeededMarker = "bipraSeeded"

func (s *Store) bipra() *mongo.Collection {
	return s.families.Database().Collection("bipra_categories")
}

func (s *Store) ensureBipraIndexes(ctx context.Context) error {
	_, err := s.bipra().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "key", Value: 1}},
		// "pkb" and "PKB" are the same code
		Options: options.Index().SetUnique(true).SetCollation(&options.Collation{Locale: "en", Strength: 2}),
	})
	return err
}

// SeedBipra adds the default categories (model.DefaultBipra) the first time
// a database is used, like SeedWorshipTypes; categories an admin deleted
// never come back.
func (s *Store) SeedBipra(ctx context.Context) error {
	if err := s.ensureBipraIndexes(ctx); err != nil {
		return err
	}
	n, err := s.meta().CountDocuments(ctx, bson.D{{Key: "_id", Value: bipraSeededMarker}})
	if err != nil || n > 0 {
		return err
	}
	if n, err = s.bipra().CountDocuments(ctx, bson.D{}); err != nil {
		return err
	}
	if n == 0 {
		for _, c := range model.DefaultBipra() {
			if _, err := s.bipra().InsertOne(ctx, c); err != nil && !mongo.IsDuplicateKeyError(err) {
				return err
			}
		}
	}
	_, err = s.meta().UpdateOne(ctx, bson.D{{Key: "_id", Value: bipraSeededMarker}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "at", Value: time.Now().UTC()}}}},
		options.UpdateOne().SetUpsert(true))
	return err
}

// BipraRules returns the categories in order. A database that was never
// seeded (e.g. a test or a fresh copy before the server started) gets the
// defaults, so members always have categories.
func (s *Store) BipraRules(ctx context.Context) (model.BipraRules, error) {
	cur, err := s.bipra().Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "order", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	rules := model.BipraRules{}
	if err := cur.All(ctx, &rules); err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		seeded, err := s.meta().CountDocuments(ctx, bson.D{{Key: "_id", Value: bipraSeededMarker}})
		if err != nil {
			return nil, err
		}
		if seeded == 0 {
			return model.DefaultBipra(), nil
		}
	}
	return rules, nil
}

// CreateBipra adds a category at the end of the list.
func (s *Store) CreateBipra(ctx context.Context, c model.BipraCategory) (model.BipraCategory, error) {
	var last model.BipraCategory
	err := s.bipra().FindOne(ctx, bson.D{}, options.FindOne().SetSort(bson.D{{Key: "order", Value: -1}})).Decode(&last)
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return c, err
	}
	c.ID = bson.NewObjectID()
	c.Order = last.Order + 1
	if _, err := s.bipra().InsertOne(ctx, c); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return c, ErrBipraExists
		}
		return c, err
	}
	return c, nil
}

// UpdateBipra replaces a category's fields (not its order).
func (s *Store) UpdateBipra(ctx context.Context, c model.BipraCategory) (model.BipraCategory, error) {
	var old model.BipraCategory
	if err := s.bipra().FindOne(ctx, bson.D{{Key: "_id", Value: c.ID}}).Decode(&old); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c, ErrNotFound
		}
		return c, err
	}
	c.Order = old.Order
	if _, err := s.bipra().ReplaceOne(ctx, bson.D{{Key: "_id", Value: c.ID}}, c); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return c, ErrBipraExists
		}
		return c, err
	}
	return c, nil
}

func (s *Store) DeleteBipra(ctx context.Context, id bson.ObjectID) error {
	res, err := s.bipra().DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderBipra sets the order to the position of each id in ids (which must
// be every category, once).
func (s *Store) ReorderBipra(ctx context.Context, ids []bson.ObjectID) error {
	n, err := s.bipra().CountDocuments(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}})
	if err != nil {
		return err
	}
	total, err := s.bipra().CountDocuments(ctx, bson.D{})
	if err != nil {
		return err
	}
	seen := map[bson.ObjectID]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	if int(n) != len(ids) || len(seen) != len(ids) || n != total {
		return ErrNotFound
	}
	for i, id := range ids {
		if _, err := s.bipra().UpdateOne(ctx, bson.D{{Key: "_id", Value: id}}, bson.D{{Key: "$set", Value: bson.D{{Key: "order", Value: i + 1}}}}); err != nil {
			return err
		}
	}
	return nil
}

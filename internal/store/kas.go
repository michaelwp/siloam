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

// kas is the kolom cash books (one collection, entries by kolom).
func (s *Store) kas() *mongo.Collection {
	return s.families.Database().Collection("kas")
}

func (s *Store) ensureKasIndexes(ctx context.Context) error {
	_, err := s.kas().Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "kolom", Value: 1}, {Key: "date", Value: 1}}},
		{Keys: bson.D{{Key: "serviceId", Value: 1}}},
	})
	return err
}

// ListKas is the cash book of kolom k, oldest first (entries of one day in
// the order they were added).
func (s *Store) ListKas(ctx context.Context, k int) ([]model.KasEntry, error) {
	cur, err := s.kas().Find(ctx, bson.D{{Key: "kolom", Value: k}},
		options.Find().SetSort(bson.D{{Key: "date", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	out := []model.KasEntry{}
	return out, cur.All(ctx, &out)
}

// KasSummaries are the totals of every kolom that has cash entries, and
// those of the entries dated from from up to (not including) to.
func (s *Store) KasSummaries(ctx context.Context, from, to time.Time) ([]model.KasSummary, error) {
	sum := func(kind string, more ...bson.D) bson.D {
		cond := bson.A{bson.D{{Key: "$eq", Value: bson.A{"$kind", kind}}}}
		for _, m := range more {
			cond = append(cond, m)
		}
		return bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{
			bson.D{{Key: "$and", Value: cond}}, "$amount", 0,
		}}}}}
	}
	inPeriod := []bson.D{
		{{Key: "$gte", Value: bson.A{"$date", from}}},
		{{Key: "$lt", Value: bson.A{"$date", to}}},
	}
	cur, err := s.kas().Aggregate(ctx, mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$kolom"},
			{Key: "in", Value: sum(model.KasIn)},
			{Key: "out", Value: sum(model.KasOut)},
			{Key: "monthIn", Value: sum(model.KasIn, inPeriod...)},
			{Key: "monthOut", Value: sum(model.KasOut, inPeriod...)},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "last", Value: bson.D{{Key: "$max", Value: "$date"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	})
	if err != nil {
		return nil, err
	}
	out := []model.KasSummary{}
	return out, cur.All(ctx, &out)
}

func (s *Store) GetKas(ctx context.Context, id bson.ObjectID) (model.KasEntry, error) {
	var e model.KasEntry
	err := s.kas().FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&e)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return e, ErrNotFound
	}
	return e, err
}

func (s *Store) CreateKas(ctx context.Context, e model.KasEntry) (model.KasEntry, error) {
	now := time.Now().UTC()
	e.ID = bson.NewObjectID()
	e.CreatedAt, e.UpdatedAt = now, now
	e.UpdatedBy = ""
	_, err := s.kas().InsertOne(ctx, e)
	return e, err
}

// UpdateKas replaces what an entry says (who added it and when stay).
func (s *Store) UpdateKas(ctx context.Context, e model.KasEntry) (model.KasEntry, error) {
	fields := bson.D{
		{Key: "kolom", Value: e.Kolom},
		{Key: "kind", Value: e.Kind},
		{Key: "date", Value: e.Date},
		{Key: "amount", Value: e.Amount},
		{Key: "description", Value: e.Description},
		{Key: "updatedBy", Value: e.UpdatedBy},
		{Key: "updatedAt", Value: time.Now().UTC()},
	}
	update := bson.D{}
	if e.ServiceID != nil {
		fields = append(fields, bson.E{Key: "serviceId", Value: e.ServiceID}, bson.E{Key: "serviceLabel", Value: e.ServiceLabel})
	} else {
		update = append(update, bson.E{Key: "$unset", Value: bson.D{{Key: "serviceId", Value: ""}, {Key: "serviceLabel", Value: ""}}})
	}
	update = append(bson.D{{Key: "$set", Value: fields}}, update...)
	var out model.KasEntry
	err := s.kas().FindOneAndUpdate(ctx, bson.D{{Key: "_id", Value: e.ID}}, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return out, ErrNotFound
	}
	return out, err
}

func (s *Store) DeleteKas(ctx context.Context, id bson.ObjectID) error {
	res, err := s.kas().DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	if err == nil && res.DeletedCount == 0 {
		return ErrNotFound
	}
	return err
}

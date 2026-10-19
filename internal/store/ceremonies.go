package store

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// In-progress WebAuthn ceremonies (challenge state between "begin" and
// "finish"). They live in MongoDB rather than in memory so a ceremony works
// when the two requests reach different app instances (e.g. two Fly.io
// machines) or the instance restarts in between. A TTL index removes
// abandoned ones.

func (s *Store) ensureCeremonyIndexes(ctx context.Context) error {
	_, err := s.ceremonies.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0),
	})
	return err
}

// PutCeremony stores ceremony state (opaque bytes) under id until expiresAt.
func (s *Store) PutCeremony(ctx context.Context, id string, data []byte, expiresAt time.Time) error {
	_, err := s.ceremonies.InsertOne(ctx, bson.D{
		{Key: "_id", Value: id},
		{Key: "data", Value: data},
		{Key: "expiresAt", Value: expiresAt},
	})
	return err
}

// TakeCeremony returns and deletes an unexpired ceremony, so each challenge
// can be used once. ErrNotFound if it is unknown, used or expired.
func (s *Store) TakeCeremony(ctx context.Context, id string, now time.Time) ([]byte, error) {
	var doc struct {
		Data []byte `bson:"data"`
	}
	err := s.ceremonies.FindOneAndDelete(ctx, bson.D{
		{Key: "_id", Value: id},
		{Key: "expiresAt", Value: bson.D{{Key: "$gt", Value: now}}},
	}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	return doc.Data, err
}

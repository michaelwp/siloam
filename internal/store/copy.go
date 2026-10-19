package store

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// CopyResult counts the documents copied per collection.
type CopyResult map[string]int

// CopyCollections are the collections CopyFrom can copy. Sessions are never
// copied: users sign in again on the new database.
var CopyCollections = []string{"families", "members", "users"}

func (s *Store) collection(name string) (*mongo.Collection, error) {
	switch name {
	case "families":
		return s.families, nil
	case "members":
		return s.members, nil
	case "users":
		return s.users, nil
	}
	return nil, fmt.Errorf("unknown collection %q (choose from %v)", name, CopyCollections)
}

// CopyFrom copies the named collections (all of CopyCollections when names is
// empty) from src into s, keeping document IDs so references between them stay
// valid. The target collections must be empty unless replace is set, in which
// case they are cleared first (and sessions too, when users are replaced).
// Collections not named are left untouched.
func (s *Store) CopyFrom(ctx context.Context, src *Store, names []string, replace bool) (CopyResult, error) {
	if len(names) == 0 {
		names = CopyCollections
	}
	same, err := sameDatabase(ctx, src, s)
	if err != nil {
		return nil, fmt.Errorf("compare source and target: %w", err)
	}
	if same {
		return nil, ErrSameDatabase
	}
	type pair struct{ src, dst *mongo.Collection }
	var pairs []pair
	for _, name := range names {
		dst, err := s.collection(name)
		if err != nil {
			return nil, err
		}
		from, _ := src.collection(name)
		n, err := dst.CountDocuments(ctx, bson.D{})
		if err != nil {
			return nil, err
		}
		if n > 0 && !replace {
			return nil, fmt.Errorf("target collection %q already has %d documents; pass -replace to overwrite", name, n)
		}
		pairs = append(pairs, pair{from, dst})
	}
	if replace {
		for _, p := range pairs {
			if _, err := p.dst.DeleteMany(ctx, bson.D{}); err != nil {
				return nil, err
			}
			if p.dst == s.users {
				if _, err := s.sessions.DeleteMany(ctx, bson.D{}); err != nil {
					return nil, err
				}
			}
		}
	}

	res := CopyResult{}
	for _, p := range pairs {
		n, err := copyCollection(ctx, p.src, p.dst)
		res[p.dst.Name()] = n
		if err != nil {
			return res, fmt.Errorf("copy %s: %w", p.dst.Name(), err)
		}
	}
	return res, nil
}

var ErrSameDatabase = errors.New("source and target are the same database")

// sameDatabase writes a random marker into dst and checks whether src sees
// it — reliable however the two URIs spell the host (localhost, 127.0.0.1,
// a container name, a replica-set seed list, …).
func sameDatabase(ctx context.Context, src, dst *Store) (bool, error) {
	const probe = "copy_probe"
	marker := bson.NewObjectID()
	dstProbe := dst.families.Database().Collection(probe)
	if _, err := dstProbe.InsertOne(ctx, bson.D{{Key: "_id", Value: marker}}); err != nil {
		return false, err
	}
	defer func() { _ = dstProbe.Drop(context.WithoutCancel(ctx)) }()
	n, err := src.families.Database().Collection(probe).CountDocuments(ctx, bson.D{{Key: "_id", Value: marker}})
	return n > 0, err
}

func copyCollection(ctx context.Context, src, dst *mongo.Collection) (int, error) {
	cur, err := src.Find(ctx, bson.D{})
	if err != nil {
		return 0, err
	}
	defer func() { _ = cur.Close(ctx) }()

	const batchSize = 500
	total := 0
	batch := make([]any, 0, batchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if _, err := dst.InsertMany(ctx, batch); err != nil {
			return err
		}
		total += len(batch)
		batch = batch[:0]
		return nil
	}
	for cur.Next(ctx) {
		// copy the raw document as-is (cur.Current is reused, so clone it)
		batch = append(batch, bson.Raw(append([]byte(nil), cur.Current...)))
		if len(batch) == batchSize {
			if err := flush(); err != nil {
				return total, err
			}
		}
	}
	if err := cur.Err(); err != nil {
		return total, err
	}
	return total, flush()
}

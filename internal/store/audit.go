package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"siloam/internal/model"
)

// AuditLabel names the thing a request is about to delete ("Kel. Umboh",
// "Kolom 3", …), so the log still says what it was afterwards. Unknown
// kinds or ids give "".
func (s *Store) AuditLabel(ctx context.Context, kind string, id bson.ObjectID) string {
	var coll *mongo.Collection
	switch kind {
	case "koloms":
		coll = s.koloms
	case "families":
		coll = s.families
	case "members":
		coll = s.members
	case "worship-types":
		coll = s.worshipTypes
	case "services":
		coll = s.services
	case "users":
		coll = s.users
	case "bipra":
		coll = s.bipra()
	case "kas":
		coll = s.kas()
	default:
		return ""
	}
	var doc struct {
		Number   int       `bson:"number"`
		Name     string    `bson:"name"`
		Username string    `bson:"username"`
		Host     string    `bson:"host"`
		Date     time.Time `bson:"date"`
		Kolom    int       `bson:"kolom"`
		Kind     string    `bson:"kind"`
		Amount   int64     `bson:"amount"`
	}
	if err := coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doc); err != nil {
		return ""
	}
	switch kind {
	case "koloms":
		return fmt.Sprintf("Kolom %d", doc.Number)
	case "families":
		return "Kel. " + doc.Name
	case "users":
		return doc.Username
	case "kas":
		return model.KasLabel(doc.Kolom, doc.Kind, doc.Amount, doc.Date.Format("2006-01-02"))
	case "services":
		label := doc.Date.Format("2006-01-02")
		if doc.Host != "" {
			label = "Kel. " + doc.Host + " · " + label
		}
		return label
	default:
		return doc.Name
	}
}

// ---- Android release (in the meta collection) ----

const androidReleaseKey = "androidRelease"

func (s *Store) meta() *mongo.Collection { return s.families.Database().Collection("meta") }

// GetAndroidRelease returns the release set on the admin page, or nil when
// none was set (the server's ANDROID_* settings then apply).
func (s *Store) GetAndroidRelease(ctx context.Context) (*model.AndroidRelease, error) {
	var r model.AndroidRelease
	err := s.meta().FindOne(ctx, bson.D{{Key: "_id", Value: androidReleaseKey}}).Decode(&r)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) SetAndroidRelease(ctx context.Context, r model.AndroidRelease) error {
	_, err := s.meta().ReplaceOne(ctx, bson.D{{Key: "_id", Value: androidReleaseKey}}, r, options.Replace().SetUpsert(true))
	return err
}

// ClearAndroidRelease removes the admin page's setting (back to ANDROID_*).
func (s *Store) ClearAndroidRelease(ctx context.Context) error {
	_, err := s.meta().DeleteOne(ctx, bson.D{{Key: "_id", Value: androidReleaseKey}})
	return err
}

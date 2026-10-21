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

const auditTTLIndex = "at_ttl"

// EnsureAuditIndexes keeps audit entries for [retention], after which MongoDB
// deletes them (a TTL index on "at"); a changed retention updates the index.
func (s *Store) EnsureAuditIndexes(ctx context.Context, retention time.Duration) error {
	secs := int32(retention / time.Second)
	_, err := s.audit.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "at", Value: 1}},
		Options: options.Index().SetName(auditTTLIndex).SetExpireAfterSeconds(secs),
	})
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) && (cmdErr.Code == 85 || cmdErr.Code == 86) { // IndexOptionsConflict / IndexKeySpecsConflict
		err = s.audit.Database().RunCommand(ctx, bson.D{
			{Key: "collMod", Value: s.audit.Name()},
			{Key: "index", Value: bson.D{{Key: "name", Value: auditTTLIndex}, {Key: "expireAfterSeconds", Value: secs}}},
		}).Err()
	}
	if err != nil {
		return fmt.Errorf("audit index: %w", err)
	}
	_, err = s.audit.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "username", Value: 1}, {Key: "_id", Value: -1}}},
		{Keys: bson.D{{Key: "category", Value: 1}, {Key: "_id", Value: -1}}},
	})
	return err
}

// InsertAudit records one activity.
func (s *Store) InsertAudit(ctx context.Context, e *model.AuditEntry) error {
	if e.ID.IsZero() {
		e.ID = bson.NewObjectID()
	}
	_, err := s.audit.InsertOne(ctx, e)
	return err
}

// AuditFilter narrows ListAudit; zero values match everything.
type AuditFilter struct {
	Username string
	Category string
	Failed   *bool         // true: only refused/failed requests; false: only successful ones
	From, To time.Time     // At in [From, To)
	Before   bson.ObjectID // the page after this entry (newest first)
	Limit    int64
}

// ListAudit returns entries newest first.
func (s *Store) ListAudit(ctx context.Context, f AuditFilter) ([]model.AuditEntry, error) {
	q := bson.D{}
	if f.Username != "" {
		q = append(q, bson.E{Key: "username", Value: f.Username})
	}
	if f.Category != "" {
		q = append(q, bson.E{Key: "category", Value: f.Category})
	}
	if f.Failed != nil {
		op := "$lt"
		if *f.Failed {
			op = "$gte"
		}
		q = append(q, bson.E{Key: "status", Value: bson.D{{Key: op, Value: 400}}})
	}
	at := bson.D{}
	if !f.From.IsZero() {
		at = append(at, bson.E{Key: "$gte", Value: f.From})
	}
	if !f.To.IsZero() {
		at = append(at, bson.E{Key: "$lt", Value: f.To})
	}
	if len(at) > 0 {
		q = append(q, bson.E{Key: "at", Value: at})
	}
	if !f.Before.IsZero() {
		q = append(q, bson.E{Key: "_id", Value: bson.D{{Key: "$lt", Value: f.Before}}})
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	cur, err := s.audit.Find(ctx, q, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	out := []model.AuditEntry{}
	return out, cur.All(ctx, &out)
}

// AuditUsernames lists the names that appear in the audit log (for its filter).
func (s *Store) AuditUsernames(ctx context.Context) ([]string, error) {
	var names []string
	if err := s.audit.Distinct(ctx, "username", bson.D{{Key: "username", Value: bson.D{{Key: "$ne", Value: ""}}}}).Decode(&names); err != nil {
		return nil, err
	}
	return names, nil
}

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
	default:
		return ""
	}
	var doc struct {
		Number   int       `bson:"number"`
		Name     string    `bson:"name"`
		Username string    `bson:"username"`
		Host     string    `bson:"host"`
		Date     time.Time `bson:"date"`
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

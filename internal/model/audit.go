package model

import (
	"errors"
	"net/url"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// AuditEntry is one user activity: a change made through the API, or a
// sign-in / sign-out. Request bodies are never recorded (no passwords).
type AuditEntry struct {
	ID       bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	At       time.Time      `bson:"at" json:"at"`
	UserID   *bson.ObjectID `bson:"userId,omitempty" json:"userId,omitempty"`
	Username string         `bson:"username" json:"username"` // for a failed sign-in: the name that was tried
	Category string         `bson:"category" json:"category"` // e.g. "anggota", see audit.Categories
	Action   string         `bson:"action" json:"action"`     // e.g. "Ubah anggota"
	Target   string         `bson:"target,omitempty" json:"target,omitempty"`
	TargetID string         `bson:"targetId,omitempty" json:"targetId,omitempty"`
	Method   string         `bson:"method" json:"method"`
	Path     string         `bson:"path" json:"path"`
	Status   int            `bson:"status" json:"status"` // HTTP status: 2xx = done
	IP       string         `bson:"ip" json:"ip"`
}

// AndroidRelease is the newest Android app that installed apps are told
// about. Both fields empty means "no update notice".
type AndroidRelease struct {
	LatestVersion string     `bson:"latestVersion" json:"latestVersion"` // the app's versionName, e.g. "1.0.1"
	DownloadURL   string     `bson:"downloadUrl" json:"downloadUrl"`     // where the APK is downloaded (https)
	UpdatedAt     *time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	UpdatedBy     string     `bson:"updatedBy,omitempty" json:"updatedBy,omitempty"`
}

var versionRe = regexp.MustCompile(`^\d+(\.\d+){0,3}$`)

// Validate checks a release: both values or neither, a plain dotted version
// and an https download link.
func (r AndroidRelease) Validate() error {
	if r.LatestVersion == "" && r.DownloadURL == "" {
		return nil
	}
	if r.LatestVersion == "" || r.DownloadURL == "" {
		return errors.New("isi versi dan tautan unduhan, atau kosongkan keduanya")
	}
	if !versionRe.MatchString(r.LatestVersion) {
		return errors.New(`versi harus seperti "1.0.1"`)
	}
	u, err := url.Parse(r.DownloadURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errors.New("tautan unduhan harus diawali https://")
	}
	return nil
}

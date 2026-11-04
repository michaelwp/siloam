package model

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
	// RoleScoped ("Pengguna – kelola kolom/BIPRA") manages some koloms
	// (User.Koloms: their families, members and every service there) and/or
	// some BIPRA categories (User.BipraIDs: the services whose jenis ibadah is
	// linked to one of them, in every kolom). At least one of either.
	RoleScoped = "scoped"
	// RoleViewer ("Pengunjung terdaftar – hanya lihat") changes no jemaat
	// data. Without koloms or categories they see the private data of the
	// whole congregation; with some (User.Koloms, User.BipraIDs), only of
	// those koloms and of those categories' members.
	RoleViewer = "viewer"
	// LegacyRoleKolom and LegacyRoleBipra were the one-kolom and
	// one-category roles (fields "kolom" and "bipraId"); the store moves them
	// to RoleScoped at startup.
	LegacyRoleKolom = "kolom"
	LegacyRoleBipra = "bipra"
)

// ValidRole reports whether role is one of the roles above (not the legacy ones).
func ValidRole(role string) bool {
	return role == RoleAdmin || role == RoleUser || role == RoleScoped || role == RoleViewer
}

// HasScope reports whether role keeps a list of koloms and BIPRA categories.
func HasScope(role string) bool { return role == RoleScoped || role == RoleViewer }

// User is an account that can sign in to the app.
type User struct {
	ID                 bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	Username           string          `bson:"username" json:"username"`
	PasswordHash       string          `bson:"passwordHash" json:"-"`
	Role               string          `bson:"role" json:"role"`
	Koloms             []int           `bson:"koloms,omitempty" json:"koloms,omitempty"`     // RoleScoped: the koloms they manage; RoleViewer: see
	BipraIDs           []bson.ObjectID `bson:"bipraIds,omitempty" json:"bipraIds,omitempty"` // RoleScoped: the categories they manage; RoleViewer: see
	MustChangePassword bool            `bson:"mustChangePassword" json:"mustChangePassword"`
	FailedLogins       int             `bson:"failedLogins" json:"failedLogins"` // consecutive wrong passwords
	LockedAt           *time.Time      `bson:"lockedAt" json:"lockedAt"`         // when the current/last lock started
	LockedUntil        *time.Time      `bson:"lockedUntil" json:"lockedUntil"`   // lock expiry; admin reset clears it
	WebAuthnID         []byte          `bson:"webauthnId" json:"-"`              // random user handle for passkeys
	CalendarToken      string          `bson:"calendarToken,omitempty" json:"-"` // secret of their calendar feed link
	Passkeys           []Passkey       `bson:"passkeys" json:"passkeys"`
	LastLoginAt        *time.Time      `bson:"lastLoginAt" json:"lastLoginAt"`
	LastLoginSource    string          `bson:"lastLoginSource,omitempty" json:"lastLoginSource,omitempty"` // LoginWeb or LoginAndroid
	CreatedAt          time.Time       `bson:"createdAt" json:"createdAt"`
	UpdatedAt          time.Time       `bson:"updatedAt" json:"updatedAt"`
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// ManagesAll reports whether the user may change data of every kolom (and
// the koloms and jenis ibadah themselves): admins and regular users.
func (u *User) ManagesAll() bool { return u.Role == RoleAdmin || u.Role == RoleUser }

// A scoped user's or viewer's koloms and BIPRA categories add up: their
// koloms' whole data (every service there, also BIPRA ones), and their
// categories' members and services in every kolom.

// ManagesKolom reports whether the user may change the data of kolom k
// (families, members, every service there) and see its private data:
// admins, regular users, and scoped users whose koloms include it.
func (u *User) ManagesKolom(k int) bool {
	return u.ManagesAll() || (u.Role == RoleScoped && k > 0 && slices.Contains(u.Koloms, k))
}

// IsViewer reports whether the user may only look (never change jemaat data).
func (u *User) IsViewer() bool { return u.Role == RoleViewer }

// SeesKolom reports whether the user sees the private data of all of kolom
// k: who manages it, or a viewer whose koloms include it (a viewer with no
// koloms or categories: every kolom).
func (u *User) SeesKolom(k int) bool {
	if u.ManagesKolom(k) {
		return true
	}
	if u.Role != RoleViewer {
		return false
	}
	if len(u.Koloms) == 0 && len(u.BipraIDs) == 0 {
		return true
	}
	return k > 0 && slices.Contains(u.Koloms, k)
}

// SeesBipra reports whether the user sees the private data of the members of
// BIPRA category id (in every kolom) because it is one of theirs.
func (u *User) SeesBipra(id bson.ObjectID) bool {
	return HasScope(u.Role) && slices.Contains(u.BipraIDs, id)
}

// ManagesService reports whether the user may change a service of jenis t in
// kolom k (0: the whole congregation): who manages kolom k (any jenis), or a
// scoped user whose categories include the one t is linked to (any kolom).
func (u *User) ManagesService(k int, t WorshipType) bool {
	if u.ManagesKolom(k) {
		return true
	}
	return u.Role == RoleScoped && t.BipraID != nil && slices.Contains(u.BipraIDs, *t.BipraID)
}

// SeesService reports whether service of jenis t in kolom k (0: the whole
// congregation) is in the user's data: everything, or for a user limited to
// some koloms and/or categories, their koloms' services (and the whole
// congregation's) plus their categories' services in every kolom.
func (u *User) SeesService(k int, t WorshipType) bool {
	if !HasScope(u.Role) || (len(u.Koloms) == 0 && len(u.BipraIDs) == 0) {
		return true
	}
	if len(u.Koloms) > 0 && (k == 0 || slices.Contains(u.Koloms, k)) {
		return true
	}
	return t.BipraID != nil && slices.Contains(u.BipraIDs, *t.BipraID)
}

// Scope says what a scoped user may change, for refusals.
func (u *User) Scope() string {
	if u.Role == RoleViewer {
		return "Akun Anda hanya dapat melihat data"
	}
	if u.Role != RoleScoped {
		return "Anda tidak dapat mengubah data ini"
	}
	var parts []string
	if len(u.Koloms) > 0 {
		ks := make([]string, len(u.Koloms))
		for i, k := range u.Koloms {
			ks[i] = fmt.Sprint(k)
		}
		parts = append(parts, "data Kolom "+strings.Join(ks, ", "))
	}
	if len(u.BipraIDs) > 0 {
		parts = append(parts, "jadwal ibadah kategori BIPRA Anda")
	}
	return "Anda hanya dapat mengelola " + strings.Join(parts, " dan ")
}

// IsLockedAt reports whether a temporary lock is in force at the given time.
func (u *User) IsLockedAt(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
}

// Where a sign-in came from: the web app, or the Android app (which sends
// the header X-Client: android).
const (
	LoginWeb     = "web"
	LoginAndroid = "android"
)

// Login is one successful sign-in, for the admins' user statistics (kept 90 days).
type Login struct {
	ID       bson.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID   bson.ObjectID   `bson:"userId" json:"userId"`
	Username string          `bson:"username" json:"username"`
	Role     string          `bson:"role" json:"role"`
	Koloms   []int           `bson:"koloms,omitempty" json:"koloms,omitempty"`
	BipraIDs []bson.ObjectID `bson:"bipraIds,omitempty" json:"bipraIds,omitempty"`
	Source   string          `bson:"source" json:"source"`
	At       time.Time       `bson:"at" json:"at"`
}

// Passkey is a registered WebAuthn credential. Credential holds the
// JSON-encoded webauthn.Credential so the library type can evolve freely.
type Passkey struct {
	ID         string     `bson:"id" json:"id"` // base64url credential ID
	Name       string     `bson:"name" json:"name"`
	Credential []byte     `bson:"credential" json:"-"`
	CreatedAt  time.Time  `bson:"createdAt" json:"createdAt"`
	LastUsedAt *time.Time `bson:"lastUsedAt" json:"lastUsedAt"`
}

// Session is a signed-in browser. ID is the SHA-256 of the cookie token, so a
// database leak does not leak usable tokens.
type Session struct {
	ID        string        `bson:"_id"`
	UserID    bson.ObjectID `bson:"userId"`
	UserAgent string        `bson:"userAgent"`
	Source    string        `bson:"source,omitempty"` // LoginWeb or LoginAndroid
	CreatedAt time.Time     `bson:"createdAt"`
	ExpiresAt time.Time     `bson:"expiresAt"`
}

package model

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
	// RoleKolom ("Pengguna – kelola data kolom") manages the data of one
	// kolom (User.Kolom) only: its families, members and services.
	RoleKolom = "kolom"
	// RoleBipra ("Pengguna – kelola data BIPRA") manages the services of one
	// BIPRA category (User.BipraID): those whose jenis ibadah is linked to it.
	RoleBipra = "bipra"
)

// ValidRole reports whether role is one of the roles above.
func ValidRole(role string) bool {
	return role == RoleAdmin || role == RoleUser || role == RoleKolom || role == RoleBipra
}

// User is an account that can sign in to the app.
type User struct {
	ID                 bson.ObjectID  `bson:"_id,omitempty" json:"id"`
	Username           string         `bson:"username" json:"username"`
	PasswordHash       string         `bson:"passwordHash" json:"-"`
	Role               string         `bson:"role" json:"role"`
	Kolom              int            `bson:"kolom,omitempty" json:"kolom,omitempty"`     // RoleKolom: the kolom they manage
	BipraID            *bson.ObjectID `bson:"bipraId,omitempty" json:"bipraId,omitempty"` // RoleBipra: the category they manage
	MustChangePassword bool           `bson:"mustChangePassword" json:"mustChangePassword"`
	FailedLogins       int            `bson:"failedLogins" json:"failedLogins"` // consecutive wrong passwords
	LockedAt           *time.Time     `bson:"lockedAt" json:"lockedAt"`         // when the current/last lock started
	LockedUntil        *time.Time     `bson:"lockedUntil" json:"lockedUntil"`   // lock expiry; admin reset clears it
	WebAuthnID         []byte         `bson:"webauthnId" json:"-"`              // random user handle for passkeys
	Passkeys           []Passkey      `bson:"passkeys" json:"passkeys"`
	LastLoginAt        *time.Time     `bson:"lastLoginAt" json:"lastLoginAt"`
	CreatedAt          time.Time      `bson:"createdAt" json:"createdAt"`
	UpdatedAt          time.Time      `bson:"updatedAt" json:"updatedAt"`
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

// ManagesAll reports whether the user may change data of every kolom (and
// the koloms and jenis ibadah themselves): admins and regular users.
func (u *User) ManagesAll() bool { return u.Role == RoleAdmin || u.Role == RoleUser }

// ManagesKolom reports whether the user may change, and see the private
// data of, kolom k.
func (u *User) ManagesKolom(k int) bool {
	return u.ManagesAll() || (u.Role == RoleKolom && u.Kolom == k)
}

// ManagesType reports whether a "kelola data BIPRA" user may change services
// of jenis ibadah t (linked to their category). Other roles: see ManagesKolom.
func (u *User) ManagesType(t WorshipType) bool {
	return u.Role == RoleBipra && u.BipraID != nil && t.BipraID != nil && *t.BipraID == *u.BipraID
}

// Scope says what a limited user may change, for refusals.
func (u *User) Scope() string {
	switch u.Role {
	case RoleKolom:
		return fmt.Sprintf("Anda hanya dapat mengelola data Kolom %d", u.Kolom)
	case RoleBipra:
		return "Anda hanya dapat mengelola jadwal ibadah kategori BIPRA Anda"
	}
	return "Anda tidak dapat mengubah data ini"
}

// IsLockedAt reports whether a temporary lock is in force at the given time.
func (u *User) IsLockedAt(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
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
	CreatedAt time.Time     `bson:"createdAt"`
	ExpiresAt time.Time     `bson:"expiresAt"`
}

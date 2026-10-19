package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// User is an account that can sign in to the app.
type User struct {
	ID                 bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Username           string        `bson:"username" json:"username"`
	PasswordHash       string        `bson:"passwordHash" json:"-"`
	Role               string        `bson:"role" json:"role"`
	MustChangePassword bool          `bson:"mustChangePassword" json:"mustChangePassword"`
	FailedLogins       int           `bson:"failedLogins" json:"failedLogins"` // consecutive wrong passwords
	LockedAt           *time.Time    `bson:"lockedAt" json:"lockedAt"`         // when the current/last lock started
	LockedUntil        *time.Time    `bson:"lockedUntil" json:"lockedUntil"`   // lock expiry; admin reset clears it
	WebAuthnID         []byte        `bson:"webauthnId" json:"-"`              // random user handle for passkeys
	Passkeys           []Passkey     `bson:"passkeys" json:"passkeys"`
	LastLoginAt        *time.Time    `bson:"lastLoginAt" json:"lastLoginAt"`
	CreatedAt          time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt          time.Time     `bson:"updatedAt" json:"updatedAt"`
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

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

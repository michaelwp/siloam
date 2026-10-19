package auth

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"siloam/internal/httpx"
	"siloam/internal/model"
	"siloam/internal/store"
)

// maxFailsPerIP failed sign-ins from one client IP pause sign-ins from it for
// the limiter window. Per-account protection is the lockout (MaxFailedLogins).
const maxFailsPerIP = 30

// loginFailed is the one response for every failed password sign-in — wrong
// password, unknown username or locked account — so it does not reveal which
// usernames exist or are locked.
func (s *Service) loginFailed(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusUnauthorized, fmt.Sprintf(
		"Username atau password salah. Setelah %d kali gagal, akun dikunci selama %s.",
		s.cfg.MaxFailedLogins, humanDuration(s.cfg.LockDuration)))
}

// humanDuration renders a lock duration in Indonesian, e.g. "3 menit".
func humanDuration(d time.Duration) string {
	switch {
	case d >= time.Hour && d%time.Hour == 0:
		return fmt.Sprintf("%d jam", int(d/time.Hour))
	case d >= time.Minute && d%time.Minute == 0:
		return fmt.Sprintf("%d menit", int(d/time.Minute))
	case d >= time.Minute:
		return fmt.Sprintf("%d menit", int(math.Ceil(d.Minutes())))
	default:
		return fmt.Sprintf("%d detik", int(math.Ceil(d.Seconds())))
	}
}

func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	ipKey := "ip:" + s.clientIP(r)
	if s.limiter.blocked(ipKey, maxFailsPerIP) {
		httpx.WriteError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}

	u, err := s.store.GetUserByUsername(r.Context(), strings.ToLower(strings.TrimSpace(in.Username)))
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		httpx.ServerError(w, err)
		return
	}
	hash := ""
	if u != nil {
		hash = u.PasswordHash
	}
	// always run bcrypt so every outcome takes the same time
	ok := checkPassword(hash, in.Password)
	locked := u != nil && u.IsLockedAt(time.Now())
	if u == nil || locked || !ok {
		s.limiter.fail(ipKey)
		if u != nil && !locked { // failures during a lock neither count nor extend it
			if err := s.recordFailure(r, u); err != nil {
				httpx.ServerError(w, err)
				return
			}
		}
		s.loginFailed(w)
		return
	}
	u, err = s.startSession(w, r, u.ID)
	if errors.Is(err, store.ErrLocked) { // locked while the password was checked
		s.loginFailed(w)
		return
	}
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}

// recordFailure counts a wrong password; if that locks the account it also
// ends all of the user's sessions.
func (s *Service) recordFailure(r *http.Request, u *model.User) error {
	now := time.Now().UTC()
	updated, err := s.store.RecordFailedLogin(r.Context(), u.ID, s.cfg.MaxFailedLogins, now, s.cfg.LockDuration)
	if err != nil || !updated.IsLockedAt(now) {
		return err
	}
	log.Printf("account %q locked for %s after %d failed sign-ins", updated.Username, s.cfg.LockDuration, updated.FailedLogins)
	return s.store.DeleteUserSessions(r.Context(), updated.ID, "")
}

// writeLocked is used where the user has already proven who they are (a valid
// passkey), so telling them the account is locked reveals nothing.
func writeLocked(w http.ResponseWriter, u *model.User) {
	left := time.Duration(0)
	if u.LockedUntil != nil {
		left = time.Until(*u.LockedUntil)
	}
	httpx.WriteError(w, http.StatusLocked, fmt.Sprintf(
		"Akun dikunci sementara karena terlalu banyak percobaan masuk yang gagal. Coba lagi dalam %s, atau hubungi admin.",
		humanDuration(max(left, time.Second))), "account_locked")
}

func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		if err := s.store.DeleteSession(r.Context(), hashToken(c.Value)); err != nil {
			httpx.ServerError(w, err)
			return
		}
	}
	s.clearCookie(w, sessionCookie, "/")
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) me(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, current(r).user)
}

// changeCredentials updates the signed-in user's username and password. It is
// the only write allowed while MustChangePassword is set.
func (s *Service) changeCredentials(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CurrentPassword string `json:"currentPassword"`
		Username        string `json:"username"`
		NewPassword     string `json:"newPassword"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	p := current(r)
	u := p.user
	if !checkPassword(u.PasswordHash, in.CurrentPassword) {
		// guessing the current password with a stolen session counts toward the lockout
		if err := s.recordFailure(r, u); err != nil {
			httpx.ServerError(w, err)
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "password saat ini salah")
		return
	}
	username, err := normalizeUsername(in.Username)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validatePassword(in.NewPassword, username); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.NewPassword == in.CurrentPassword {
		httpx.WriteError(w, http.StatusBadRequest, "password baru harus berbeda dari password saat ini")
		return
	}
	hash, err := hashPassword(in.NewPassword)
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	updated, err := s.store.SetCredentials(r.Context(), u.ID, username, hash)
	if err != nil {
		writeUserError(w, err)
		return
	}
	// sign out other browsers that may know the old password
	if err := s.store.DeleteUserSessions(r.Context(), u.ID, p.session.ID); err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

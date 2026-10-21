package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/httpx"
	"siloam/internal/model"
	"siloam/internal/store"
)

// Admin-only user management. The middleware has already checked the role.

// roleKolom checks a role and its kolom: a "kelola data kolom" user needs an
// existing kolom; other roles have none (0).
func (s *Service) roleKolom(r *http.Request, role string, kolom int) (int, error) {
	if !model.ValidRole(role) {
		return 0, errors.New("role tidak valid")
	}
	if role != model.RoleKolom {
		return 0, nil
	}
	if kolom <= 0 {
		return 0, errors.New("pilih kolom yang dikelola pengguna ini")
	}
	ok, err := s.store.KolomExists(r.Context(), kolom)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("kolom %d tidak ada", kolom)
	}
	return kolom, nil
}

func (s *Service) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, users)
}

// createUser adds an account with a temporary password that must be changed
// on first sign-in.
func (s *Service) createUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Kolom    int    `json:"kolom"` // for the "kolom" role
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	username, err := normalizeUsername(in.Username)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validatePassword(in.Password, username); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Role == "" {
		in.Role = model.RoleUser
	}
	kolom, err := s.roleKolom(r, in.Role, in.Kolom)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	u := &model.User{
		Username:           username,
		PasswordHash:       hash,
		Role:               in.Role,
		Kolom:              kolom,
		MustChangePassword: true,
		WebAuthnID:         randomBytes(32),
	}
	if err := s.store.CreateUser(r.Context(), u); err != nil {
		writeUserError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, u)
}

func (s *Service) targetUser(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	id, err := bson.ObjectIDFromHex(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "id tidak valid")
		return nil, false
	}
	u, err := s.store.GetUser(r.Context(), id)
	if err != nil {
		writeUserError(w, err)
		return nil, false
	}
	return u, true
}

// updateUser changes username and/or role (and a kolom user's kolom).
func (s *Service) updateUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		Kolom    int    `json:"kolom"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u, ok := s.targetUser(w, r)
	if !ok {
		return
	}
	if in.Username != "" {
		username, err := normalizeUsername(in.Username)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if username != u.Username {
			if u, err = s.store.SetUsername(r.Context(), u.ID, username); err != nil {
				writeUserError(w, err)
				return
			}
		}
	}
	if in.Role != "" && (in.Role != u.Role || in.Kolom != u.Kolom) {
		kolom, err := s.roleKolom(r, in.Role, in.Kolom)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if in.Role != u.Role || kolom != u.Kolom {
			if u.ID == current(r).user.ID {
				httpx.WriteError(w, http.StatusBadRequest, "tidak dapat mengubah role akun sendiri")
				return
			}
			if u, err = s.store.SetRole(r.Context(), u.ID, in.Role, kolom); err != nil {
				writeUserError(w, err)
				return
			}
		}
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}

// resetPassword sets a temporary password and unlocks the account; the user
// must change it on next sign-in and is signed out everywhere. Optionally
// removes their passkeys (e.g. a lost phone).
func (s *Service) resetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password      string `json:"password"`
		ClearPasskeys bool   `json:"clearPasskeys"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u, ok := s.targetUser(w, r)
	if !ok {
		return
	}
	if err := validatePassword(in.Password, u.Username); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	updated, err := s.store.ResetPassword(r.Context(), u.ID, hash, in.ClearPasskeys)
	if err != nil {
		writeUserError(w, err)
		return
	}
	keep := ""
	if p := current(r); u.ID == p.user.ID {
		keep = p.session.ID
	}
	if err := s.store.DeleteUserSessions(r.Context(), u.ID, keep); err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

func (s *Service) deleteUser(w http.ResponseWriter, r *http.Request) {
	u, ok := s.targetUser(w, r)
	if !ok {
		return
	}
	if u.ID == current(r).user.ID {
		httpx.WriteError(w, http.StatusBadRequest, "tidak dapat menghapus akun sendiri")
		return
	}
	if err := s.store.DeleteUser(r.Context(), u.ID); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "pengguna tidak ditemukan")
	case errors.Is(err, store.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "username sudah dipakai")
	case errors.Is(err, store.ErrLastAdmin):
		httpx.WriteError(w, http.StatusBadRequest, "harus ada minimal satu admin")
	default:
		httpx.ServerError(w, err)
	}
}

// ResetUser is the command-line recovery path (e.g. the only admin is locked
// out): it unlocks the account, sets a random temporary password that must be
// changed at next sign-in, ends its sessions and returns the password.
func (s *Service) ResetUser(ctx context.Context, username string) (string, error) {
	name, err := normalizeUsername(username)
	if err != nil {
		return "", err
	}
	u, err := s.store.GetUserByUsername(ctx, name)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", fmt.Errorf("user %q not found", name)
		}
		return "", err
	}
	password := generatePassword(14)
	hash, err := hashPassword(password)
	if err != nil {
		return "", err
	}
	if _, err := s.store.ResetPassword(ctx, u.ID, hash, false); err != nil {
		return "", err
	}
	return password, s.store.DeleteUserSessions(ctx, u.ID, "")
}

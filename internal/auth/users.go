package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"siloam/internal/audit"
	"siloam/internal/httpx"
	"siloam/internal/model"
	"siloam/internal/store"
)

// Admin-only user management. The middleware has already checked the role.

// roleScope checks a role and what it is limited to: koloms and/or BIPRA
// categories; a scoped user needs at least one, a viewer any number (none:
// everything), each existing; other roles have neither.
// The koloms come back sorted and both lists without duplicates.
func (s *Service) roleScope(r *http.Request, role string, koloms []int, bipras []string) ([]int, []bson.ObjectID, error) {
	if !model.ValidRole(role) {
		return nil, nil, errors.New("role tidak valid")
	}
	if !model.HasScope(role) {
		return nil, nil, nil
	}
	// a viewer without any sees everything
	if role == model.RoleScoped && len(koloms) == 0 && len(bipras) == 0 {
		return nil, nil, errors.New("pilih minimal satu kolom atau kategori BIPRA yang dikelola pengguna ini")
	}
	// a treasurer keeps the kas of koloms and/or categories
	if role == model.RoleTreasurer && len(koloms) == 0 && len(bipras) == 0 {
		return nil, nil, errors.New("pilih minimal satu kolom atau kategori BIPRA yang kasnya dikelola bendahara ini")
	}
	var ks []int
	for _, k := range koloms {
		if k <= 0 {
			return nil, nil, errors.New("kolom tidak valid")
		}
		ok, err := s.store.KolomExists(r.Context(), k)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, fmt.Errorf("kolom %d tidak ada", k)
		}
		ks = append(ks, k)
	}
	slices.Sort(ks)
	ks = slices.Compact(ks)
	var ids []bson.ObjectID
	for _, v := range bipras {
		id, err := bson.ObjectIDFromHex(v)
		if err != nil {
			return nil, nil, errors.New("kategori BIPRA tidak valid")
		}
		if _, err := s.store.BipraCategory(r.Context(), id); err != nil {
			return nil, nil, errors.New("kategori BIPRA tidak ada")
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ks, ids, nil
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
		Username string   `json:"username"`
		Password string   `json:"password"`
		Role     string   `json:"role"`
		Koloms   []int    `json:"koloms"`   // for the "scoped" and "viewer" roles
		BipraIDs []string `json:"bipraIds"` // for the "scoped" and "viewer" roles
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
	koloms, bipras, err := s.roleScope(r, in.Role, in.Koloms, in.BipraIDs)
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
		Koloms:             koloms,
		BipraIDs:           bipras,
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

// updateUser changes username and/or role (and a scoped user's koloms and
// categories).
func (s *Service) updateUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string   `json:"username"`
		Role     string   `json:"role"`
		Koloms   []int    `json:"koloms"`
		BipraIDs []string `json:"bipraIds"`
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
	if in.Role != "" {
		koloms, bipras, err := s.roleScope(r, in.Role, in.Koloms, in.BipraIDs)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if in.Role != u.Role || !slices.Equal(koloms, u.Koloms) || !slices.Equal(bipras, u.BipraIDs) {
			if u.ID == current(r).user.ID {
				httpx.WriteError(w, http.StatusBadRequest, "tidak dapat mengubah role akun sendiri")
				return
			}
			if u, err = s.store.SetRole(r.Context(), u.ID, in.Role, koloms, bipras); err != nil {
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

// calendarToken returns the secret token of the user's calendar feed link,
// creating it on first use; {"reset": true} replaces it (the old link stops
// working).
func (s *Service) calendarToken(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Reset bool `json:"reset"`
	}
	if r.ContentLength != 0 && !httpx.Decode(w, r, &in) {
		return
	}
	u := current(r).user
	token := base64.RawURLEncoding.EncodeToString(randomBytes(24))
	var err error
	if in.Reset {
		u, err = s.store.SetCalendarToken(r.Context(), u.ID, token)
	} else {
		u, err = s.store.EnsureCalendarToken(r.Context(), u.ID, token)
	}
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"token": u.CalendarToken})
}

// logoutUser ends every session of a user (web and Android) — e.g. a lost
// phone or a shared computer. The admin must give the reason, which the
// activity log keeps. The user can sign in again.
func (s *Service) logoutUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Reason string `json:"reason"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	reason := strings.Join(strings.Fields(in.Reason), " ")
	if len([]rune(reason)) < 3 {
		httpx.WriteError(w, http.StatusBadRequest, "alasan wajib diisi")
		return
	}
	if len([]rune(reason)) > 200 {
		httpx.WriteError(w, http.StatusBadRequest, "alasan terlalu panjang (maks. 200 karakter)")
		return
	}
	u, ok := s.targetUser(w, r)
	if !ok {
		return
	}
	if u.ID == current(r).user.ID {
		httpx.WriteError(w, http.StatusBadRequest, "untuk akun sendiri gunakan Keluar")
		return
	}
	audit.SetDetail(r, "Alasan: "+reason)
	if err := s.store.DeleteUserSessions(r.Context(), u.ID, ""); err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}

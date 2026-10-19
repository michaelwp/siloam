package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"siloam/internal/httpx"
	"siloam/internal/model"
	"siloam/internal/store"
)

// waUser adapts model.User to webauthn.User.
type waUser struct {
	u     *model.User
	creds []webauthn.Credential
}

func newWAUser(u *model.User) (*waUser, error) {
	wu := &waUser{u: u}
	for _, pk := range u.Passkeys {
		var c webauthn.Credential
		if err := json.Unmarshal(pk.Credential, &c); err != nil {
			return nil, err
		}
		wu.creds = append(wu.creds, c)
	}
	return wu, nil
}

func (w *waUser) WebAuthnID() []byte                         { return w.u.WebAuthnID }
func (w *waUser) WebAuthnName() string                       { return w.u.Username }
func (w *waUser) WebAuthnDisplayName() string                { return w.u.Username }
func (w *waUser) WebAuthnCredentials() []webauthn.Credential { return w.creds }

func credentialID(c *webauthn.Credential) string {
	return base64.RawURLEncoding.EncodeToString(c.ID)
}

const ceremonyTTL = 5 * time.Minute

// beginCeremony stores the ceremony and sets its cookie.
func (s *Service) beginCeremony(w http.ResponseWriter, r *http.Request, data *webauthn.SessionData) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	id := base64.RawURLEncoding.EncodeToString(randomBytes(24))
	if err := s.store.PutCeremony(r.Context(), id, b, time.Now().Add(ceremonyTTL)); err != nil {
		return err
	}
	s.setCookie(w, ceremonyCookie, id, "/api/auth", ceremonyTTL)
	return nil
}

// finishCeremony returns the ceremony started by this browser (once).
func (s *Service) finishCeremony(w http.ResponseWriter, r *http.Request) (webauthn.SessionData, bool) {
	s.clearCookie(w, ceremonyCookie, "/api/auth")
	var data webauthn.SessionData
	c, err := r.Cookie(ceremonyCookie)
	if err != nil {
		return data, false
	}
	b, err := s.store.TakeCeremony(r.Context(), c.Value, time.Now())
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			log.Printf("passkey ceremony: %v", err)
		}
		return data, false
	}
	return data, json.Unmarshal(b, &data) == nil
}

// ---- sign in with a passkey (usernameless / discoverable) ----

func (s *Service) passkeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	// request rate is limited by RateLimit (auth bucket); this blocks IPs with many failures
	if s.limiter.blocked("ip:"+s.clientIP(r), maxFailsPerIP) {
		httpx.WriteError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}
	assertion, data, err := s.webauthn.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	if err := s.beginCeremony(w, r, data); err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, assertion)
}

func (s *Service) passkeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	data, ok := s.finishCeremony(w, r)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "sesi passkey kedaluwarsa, silakan coba lagi")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

	var found *model.User
	handler := func(_, userHandle []byte) (webauthn.User, error) {
		u, err := s.store.GetUserByWebAuthnID(r.Context(), userHandle)
		if err != nil {
			return nil, err
		}
		found = u
		return newWAUser(u)
	}
	_, cred, err := s.webauthn.FinishPasskeyLogin(handler, data, r)
	if err != nil || found == nil {
		log.Printf("passkey login failed: %v", err)
		s.limiter.fail("ip:" + s.clientIP(r))
		httpx.WriteError(w, http.StatusUnauthorized, "passkey tidak dikenali")
		return
	}
	if found.IsLockedAt(time.Now()) {
		writeLocked(w, found)
		return
	}

	// store the updated sign counter / flags; if an admin removed this passkey
	// in the meantime, the sign-in is refused
	b, err := json.Marshal(cred)
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	if err := s.store.UsePasskey(r.Context(), found.ID, credentialID(cred), b); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.WriteError(w, http.StatusUnauthorized, "passkey tidak dikenali")
			return
		}
		httpx.ServerError(w, err)
		return
	}
	u, err := s.startSession(w, r, found.ID)
	if errors.Is(err, store.ErrLocked) {
		if latest, gerr := s.store.GetUser(r.Context(), found.ID); gerr == nil {
			found = latest
		}
		writeLocked(w, found)
		return
	}
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, u)
}

// ---- manage own passkeys (signed in) ----

func (s *Service) passkeyRegisterBegin(w http.ResponseWriter, r *http.Request) {
	wu, err := newWAUser(current(r).user)
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	exclude := make([]protocol.CredentialDescriptor, 0, len(wu.creds))
	for i := range wu.creds {
		exclude = append(exclude, wu.creds[i].Descriptor())
	}
	creation, data, err := s.webauthn.BeginRegistration(wu,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(exclude),
	)
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	if err := s.beginCeremony(w, r, data); err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, creation)
}

func (s *Service) passkeyRegisterFinish(w http.ResponseWriter, r *http.Request) {
	data, ok := s.finishCeremony(w, r)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "sesi passkey kedaluwarsa, silakan coba lagi")
		return
	}
	u := current(r).user
	wu, err := newWAUser(u)
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	cred, err := s.webauthn.FinishRegistration(wu, data, r)
	if err != nil {
		log.Printf("passkey registration failed: %v", err)
		httpx.WriteError(w, http.StatusBadRequest, "pendaftaran passkey gagal")
		return
	}
	b, err := json.Marshal(cred)
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	updated, err := s.store.AddPasskey(r.Context(), u.ID, model.Passkey{
		ID:         credentialID(cred),
		Name:       passkeyName(r.URL.Query().Get("name"), len(u.Passkeys)+1),
		Credential: b,
		CreatedAt:  time.Now().UTC(),
	})
	if err != nil {
		httpx.ServerError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, updated)
}

func passkeyName(name string, n int) string {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return "Passkey " + strconv.Itoa(n)
	}
	if r := []rune(name); len(r) > 64 {
		name = string(r[:64])
	}
	return name
}

func (s *Service) passkeyRename(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u := current(r).user
	i := slices.IndexFunc(u.Passkeys, func(p model.Passkey) bool { return p.ID == r.PathValue("id") })
	updated, err := s.store.RenamePasskey(r.Context(), u.ID, r.PathValue("id"), passkeyName(in.Name, i+1))
	writePasskeyResult(w, updated, err)
}

func (s *Service) passkeyDelete(w http.ResponseWriter, r *http.Request) {
	updated, err := s.store.DeletePasskey(r.Context(), current(r).user.ID, r.PathValue("id"))
	writePasskeyResult(w, updated, err)
}

func writePasskeyResult(w http.ResponseWriter, u *model.User, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "passkey tidak ditemukan")
	case err != nil:
		httpx.ServerError(w, err)
	default:
		httpx.WriteJSON(w, http.StatusOK, u)
	}
}

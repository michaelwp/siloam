// Package httpx holds small JSON helpers shared by the HTTP handlers.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes {"error": msg}; code is an optional machine-readable reason.
func WriteError(w http.ResponseWriter, status int, msg string, code ...string) {
	body := map[string]string{"error": msg}
	if len(code) > 0 {
		body["code"] = code[0]
	}
	WriteJSON(w, status, body)
}

func ServerError(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	WriteError(w, http.StatusInternalServerError, "terjadi kesalahan server")
}

// Decode reads a JSON body (max 1 MiB); on failure it writes a 400 and returns false.
func Decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		WriteError(w, http.StatusBadRequest, "data tidak valid: "+err.Error())
		return false
	}
	return true
}

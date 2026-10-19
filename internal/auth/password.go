package auth

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Password rules. Keep in sync with web/src/password.ts, which shows the same
// checklist live while typing.

const (
	minPasswordLen = 10
	maxPasswordLen = 72 // bcrypt limit (bytes)
)

// commonPasswordWords are rejected when a password is just one of them plus
// digits/symbols (e.g. "Password123!", "Jemaat2027#").
var commonPasswordWords = map[string]bool{
	"password": true, "passwrd": true, "pass": true, "admin": true, "administrator": true,
	"qwerty": true, "qwertyuiop": true, "asdf": true, "asdfgh": true, "zxcvbn": true,
	"letmein": true, "welcome": true, "iloveyou": true, "monkey": true, "dragon": true,
	"sunshine": true, "princess": true, "football": true, "master": true, "secret": true,
	"rahasia": true, "sandi": true, "katasandi": true, "indonesia": true, "manado": true,
	"jemaat": true, "kolom": true, "gereja": true, "siloam": true, "tuhan": true,
	"yesus": true, "jesus": true, "haleluya": true, "amin": true,
}

type passwordRule struct {
	msg string
	ok  func(p, username string) bool
}

func hasRune(p string, f func(rune) bool) bool { return strings.ContainsFunc(p, f) }

func isSymbol(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) }

func lettersOnly(p string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, p)
}

var passwordRules = []passwordRule{
	{"minimal 10 karakter", func(p, _ string) bool { return utf8.RuneCountInString(p) >= minPasswordLen }},
	{"mengandung huruf besar dan huruf kecil", func(p, _ string) bool {
		return hasRune(p, unicode.IsUpper) && hasRune(p, unicode.IsLower)
	}},
	{"mengandung angka", func(p, _ string) bool { return hasRune(p, unicode.IsDigit) }},
	{"mengandung simbol (mis. ! @ # ?)", func(p, _ string) bool { return hasRune(p, isSymbol) }},
	{"tidak mengandung username", func(p, username string) bool {
		u := strings.ToLower(strings.TrimSpace(username))
		return len(u) < 3 || !strings.Contains(strings.ToLower(p), u)
	}},
	{"bukan password yang umum dipakai", func(p, _ string) bool { return !commonPasswordWords[lettersOnly(p)] }},
}

// validatePassword checks p against all rules and lists every rule it misses.
func validatePassword(p, username string) error {
	if len(p) > maxPasswordLen {
		return errors.New("password terlalu panjang: maksimal 72 byte (huruf beraksen dan emoji dihitung lebih dari satu)")
	}
	var missing []string
	for _, r := range passwordRules {
		if !r.ok(p, username) {
			missing = append(missing, r.msg)
		}
	}
	if len(missing) > 0 {
		return errors.New("password harus " + strings.Join(missing, ", "))
	}
	return nil
}

// generatePassword returns a random password that satisfies every rule
// (one of each character class, no look-alike characters).
func generatePassword(length int) string {
	sets := []string{"ABCDEFGHJKLMNPQRSTUVWXYZ", "abcdefghijkmnpqrstuvwxyz", "23456789", "!@#$%&*?-_+="}
	all := strings.Join(sets, "")
	out := make([]byte, 0, length)
	for _, set := range sets {
		out = append(out, set[randomInt(len(set))])
	}
	for len(out) < length {
		out = append(out, all[randomInt(len(all))])
	}
	for i := len(out) - 1; i > 0; i-- {
		j := randomInt(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

func randomInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}
	return int(v.Int64())
}

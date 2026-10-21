package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAndroidReleaseValidate(t *testing.T) {
	for _, c := range []struct {
		r  AndroidRelease
		ok bool
	}{
		{AndroidRelease{}, true}, // not configured
		{AndroidRelease{"1.0.1", "https://example.org/app.apk"}, true},
		{AndroidRelease{"2", "https://example.org/app.apk"}, true},
		{AndroidRelease{"1.0.1", ""}, false},
		{AndroidRelease{"", "https://example.org/app.apk"}, false},
		{AndroidRelease{"v1.0.1", "https://example.org/app.apk"}, false},
		{AndroidRelease{"1.0.1-beta", "https://example.org/app.apk"}, false},
		{AndroidRelease{"1.0.1", "http://example.org/app.apk"}, false},
		{AndroidRelease{"1.0.1", "example.org/app.apk"}, false},
	} {
		if err := c.r.Validate(); (err == nil) != c.ok {
			t.Errorf("%+v: err = %v, want ok=%v", c.r, err, c.ok)
		}
	}
}

func TestAndroidReleaseEndpoint(t *testing.T) {
	a := &API{Android: AndroidRelease{"1.0.1", "https://example.org/app.apk"}}
	mux := http.NewServeMux()
	a.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/app/android", nil))
	var got AndroidRelease
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != 200 || got != a.Android {
		t.Fatalf("%d %s (%v)", rec.Code, rec.Body, err)
	}
}

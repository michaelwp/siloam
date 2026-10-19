package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHandler(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":       {Data: []byte("<!doctype html>app")},
		"favicon.svg":      {Data: []byte("<svg/>")},
		"assets/app-1.js":  {Data: []byte("console.log(1)")},
		"assets/app-1.css": {Data: []byte("body{}")},
	}
	h := spaHandler(dist)
	cases := []struct {
		path      string
		status    int
		body      string
		immutable bool
	}{
		{"/", 200, "app", false},
		{"/keluarga", 200, "app", false},               // client-side route
		{"/assets/app-1.js", 200, "console.log", true}, // hashed asset
		{"/assets/app-OLD.js", 404, "", false},         // stale asset: never index.html
		{"/assets/", 404, "", false},                   // no directory listing
		{"/assets", 404, "", false},
		{"/favicon.svg", 200, "<svg/>", false},
		{"/index.html", 200, "app", false},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		cache := rec.Header().Get("Cache-Control")
		if rec.Code != c.status || !strings.Contains(rec.Body.String(), c.body) || strings.Contains(cache, "immutable") != c.immutable {
			t.Errorf("%s: status=%d cache=%q body=%.30q", c.path, rec.Code, cache, rec.Body.String())
		}
		if c.status == 200 && c.body == "app" && cache != "no-cache" {
			t.Errorf("%s: index.html must not be cached, got %q", c.path, cache)
		}
	}
}

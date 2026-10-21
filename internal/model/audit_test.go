package model

import "testing"

func TestAndroidReleaseValidate(t *testing.T) {
	for _, c := range []struct {
		r  AndroidRelease
		ok bool
	}{
		{AndroidRelease{}, true}, // off
		{AndroidRelease{LatestVersion: "1.0.1", DownloadURL: "https://example.org/app.apk"}, true},
		{AndroidRelease{LatestVersion: "2", DownloadURL: "https://example.org/app.apk"}, true},
		{AndroidRelease{LatestVersion: "1.0.1"}, false},
		{AndroidRelease{DownloadURL: "https://example.org/app.apk"}, false},
		{AndroidRelease{LatestVersion: "v1.0.1", DownloadURL: "https://example.org/app.apk"}, false},
		{AndroidRelease{LatestVersion: "1.0.1-beta", DownloadURL: "https://example.org/app.apk"}, false},
		{AndroidRelease{LatestVersion: "1.0.1", DownloadURL: "http://example.org/app.apk"}, false},
		{AndroidRelease{LatestVersion: "1.0.1", DownloadURL: "example.org/app.apk"}, false},
	} {
		if err := c.r.Validate(); (err == nil) != c.ok {
			t.Errorf("%+v: err = %v, want ok=%v", c.r, err, c.ok)
		}
	}
}

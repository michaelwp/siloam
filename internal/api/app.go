package api

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
)

// AndroidRelease is the newest Android app the server tells installed apps
// about (ANDROID_LATEST_VERSION / ANDROID_DOWNLOAD_URL). The zero value means
// "not configured": the app then never says an update is available.
type AndroidRelease struct {
	LatestVersion string `json:"latestVersion"` // the app's versionName, e.g. "1.0.1"
	DownloadURL   string `json:"downloadUrl"`   // where the APK is downloaded (https)
}

var versionRe = regexp.MustCompile(`^\d+(\.\d+){0,3}$`)

// Validate checks a configured release: both values or neither, a plain
// dotted version, and an https download link.
func (r AndroidRelease) Validate() error {
	if r.LatestVersion == "" && r.DownloadURL == "" {
		return nil
	}
	if r.LatestVersion == "" || r.DownloadURL == "" {
		return errors.New("set both ANDROID_LATEST_VERSION and ANDROID_DOWNLOAD_URL, or neither")
	}
	if !versionRe.MatchString(r.LatestVersion) {
		return errors.New(`ANDROID_LATEST_VERSION must look like "1.0.1"`)
	}
	u, err := url.Parse(r.DownloadURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errors.New("ANDROID_DOWNLOAD_URL must be an https:// link")
	}
	return nil
}

// androidRelease answers the app's update check (public).
func (a *API) androidRelease(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.Android)
}

//go:build windows

// Package dart registers the Dart SDK with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package dart

import (
	"encoding/json"
	"fmt"
	"net/http"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Dart", packages.DownloadSpec{
		Version:    "3.7.2",
		URL:        "https://storage.googleapis.com/dart-archive/channels/stable/release/3.7.2/sdk/dartsdk-windows-x64-release.zip",
		FileName:   "dartsdk-windows-x64-release.zip",
		InstallDir: "bin/dart",
		StripTop:   "dart-sdk/",
		Kind:       packages.Zip,
		CheckFile:  "bin/dart.exe",
		Notes:      "Dart SDK — optimised for client + server; use with Flutter.",
	}, resolveDartLatest)
}

func resolveDartLatest(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	const apiURL = "https://storage.googleapis.com/dart-archive/channels/stable/release/latest/VERSION"
	log("  resolving latest stable Dart SDK ...")
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", "", "", err
	}
	req.Header.Set("User-Agent", "GoAMPP")
	resp, err := packages.LatestHTTPClient.Do(req)
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", "", "", fmt.Errorf("Dart archive returned HTTP %d", resp.StatusCode)
	}
	var data struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", "", "", err
	}
	if data.Version == "" {
		return "", "", "", "", fmt.Errorf("Dart stable version missing")
	}
	file := "dartsdk-windows-x64-release.zip"
	dlURL := fmt.Sprintf("https://storage.googleapis.com/dart-archive/channels/stable/release/%s/sdk/%s", data.Version, file)
	return dlURL, file, "dart-sdk/", data.Version, nil
}

//go:build windows

// Package golang registers the Go toolchain with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package golang

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Go", packages.DownloadSpec{
		Version:    "1.27.1",
		URL:        "https://go.dev/dl/go1.27.1.windows-amd64.zip",
		FileName:   "go1.27.1.windows-amd64.zip",
		InstallDir: "bin/go",
		StripTop:   "go/",
		Kind:       packages.Zip,
		CheckFile:  "bin/go.exe",
		Notes:      "Go compiler + standard tooling. Use `go run`, `go build`, `go mod`.",
	}, resolveGoLatest)
}

func resolveGoLatest(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	const apiURL = "https://go.dev/dl/?mode=json"
	log("  resolving latest stable Go Windows amd64 archive ...")
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
		return "", "", "", "", fmt.Errorf("Go download API returned HTTP %d", resp.StatusCode)
	}
	var releases []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
		Files   []struct {
			Filename string `json:"filename"`
			OS       string `json:"os"`
			Arch     string `json:"arch"`
			Kind     string `json:"kind"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", "", "", "", err
	}
	for _, release := range releases {
		if !release.Stable {
			continue
		}
		for _, file := range release.Files {
			if file.OS == "windows" && file.Arch == "amd64" && file.Kind == "archive" {
				v := strings.TrimPrefix(release.Version, "go")
				return "https://go.dev/dl/" + file.Filename, file.Filename, "go/", v, nil
			}
		}
	}
	return "", "", "", "", fmt.Errorf("stable Go windows-amd64 archive not found")
}

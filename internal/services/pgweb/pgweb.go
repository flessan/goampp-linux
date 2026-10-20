//go:build windows

// Package pgweb registers pgweb with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package pgweb

import (
	"regexp"

	"goampp/internal/packages"
)

func init() {
	packages.Register("pgweb", packages.DownloadSpec{

		Version:    "0.16.2",
		URL:        "https://github.com/sosedoff/pgweb/releases/download/v0.16.2/pgweb_windows_amd64.zip",
		FileName:   "pgweb_windows_amd64.zip",
		InstallDir: "bin/pgweb",
		Kind:       packages.Zip,
		CheckFile:  "pgweb.exe",
		Notes:      "Modern PostgreSQL web client — runs at http://localhost:8081 once started.",
	}, packages.ResolveGitHubLatestAsset(
		"sosedoff", "pgweb",
		regexp.MustCompile(`^pgweb_windows_amd64\.zip$`),
	))
}

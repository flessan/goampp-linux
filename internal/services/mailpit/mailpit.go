//go:build windows

// Package mailpit registers Mailpit with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package mailpit

import (
	"regexp"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Mailpit", packages.DownloadSpec{

		Version:    "1.30.0",
		URL:        "https://github.com/axllent/mailpit/releases/download/v1.30.0/mailpit-windows-amd64.zip",
		FileName:   "mailpit-windows-amd64.zip",
		InstallDir: "bin/mailpit",
		StripTop:   "",
		Kind:       packages.Zip,
		CheckFile:  "mailpit.exe",
		Notes:      "Email testing — SMTP :1025, web UI at http://localhost:8025.",
	}, packages.ResolveGitHubLatestAsset(
		"axllent", "mailpit",
		regexp.MustCompile(`^mailpit-windows-amd64\.zip$`),
	))
}

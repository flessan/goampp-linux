//go:build windows

// Package redis registers Redis with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package redis

import (
	"regexp"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Redis", packages.DownloadSpec{
		Version:    "5.0.14.1 (tporadowski)",
		URL:        "https://github.com/tporadowski/redis/releases/download/v5.0.14.1/Redis-x64-5.0.14.1.zip",
		FileName:   "Redis-x64-5.0.14.1.zip",
		InstallDir: "bin/redis",
		StripTop:   "",
		Kind:       packages.Zip,
		CheckFile:  "redis-server.exe",
		Notes:      "Port by Tomasz Poradowski — the Microsoft fork is abandoned.",
	}, packages.ResolveGitHubLatestAsset(
		"tporadowski", "redis",
		regexp.MustCompile(`(?i)^Redis-x64-[0-9.]+\.zip$`),
	))
}

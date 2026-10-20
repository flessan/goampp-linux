//go:build windows

// Package crystal registers Crystal with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package crystal

import (
	"regexp"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Crystal", packages.DownloadSpec{

		Version:    "1.15.1",
		URL:        "https://github.com/crystal-lang/crystal/releases/download/1.15.1/crystal-1.15.1-windows-x86_64-msvc-unsupported.zip",
		FileName:   "crystal-1.15.1-windows-x86_64-msvc-unsupported.zip",
		InstallDir: "bin/crystal",
		StripTop:   "crystal-1.15.1-windows-x86_64-msvc-unsupported/",
		Kind:       packages.Zip,
		CheckFile:  "crystal.exe",
		Notes:      "Crystal 1.15.1 — experimental Windows build; statically-typed Ruby-like language.",
	}, packages.WithStripTop(
		packages.ResolveGitHubLatestAsset(
			"crystal-lang", "crystal",
			regexp.MustCompile(`^crystal-.*-windows-x86_64-msvc.*\.zip$`),
		),
		func(fileName, _ string) string { return strings.TrimSuffix(fileName, ".zip") + "/" },
	))
}

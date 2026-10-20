//go:build windows

// Package kotlin registers the Kotlin compiler with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package kotlin

import (
	"regexp"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Kotlin", packages.DownloadSpec{
		Version:    "2.1.21",
		URL:        "https://github.com/JetBrains/kotlin/releases/download/v2.1.21/kotlin-compiler-2.1.21.zip",
		FileName:   "kotlin-compiler-2.1.21.zip",
		InstallDir: "bin/kotlin",
		StripTop:   "kotlinc/",
		Kind:       packages.Zip,
		CheckFile:  "bin/kotlinc.bat",
		Notes:      "Kotlin compiler — requires Java (install Java card first).",
	}, packages.WithStripTop(
		packages.ResolveGitHubLatestAsset(
			"JetBrains", "kotlin",
			regexp.MustCompile(`^kotlin-compiler-[0-9][0-9A-Za-z.\-]*\.zip$`),
		),
		func(_, _ string) string { return "kotlinc/" },
	))
}

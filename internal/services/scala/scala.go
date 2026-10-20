//go:build windows

// Package scala registers Scala with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package scala

import (
	"regexp"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Scala", packages.DownloadSpec{
		Version:    "2.13.16",
		URL:        "https://github.com/scala/scala/releases/download/v2.13.16/scala-2.13.16.zip",
		FileName:   "scala-2.13.16.zip",
		InstallDir: "bin/scala",
		StripTop:   "scala-2.13.16/",
		Kind:       packages.Zip,
		CheckFile:  "bin/scala.bat",
		Notes:      "Scala 2.13 LTS — requires Java (install Java card first).",
	}, resolveScala213Latest)
}

// Lightbend no longer serves Scala downloads (403); the scala/scala GitHub
// releases carry the same zip artifacts.
var resolveScala213Latest = packages.WithStripTop(
	packages.ResolveGitHubSeriesAsset("scala", "scala", "v2.13.", regexp.MustCompile(`^scala-2\.13\.[0-9]+\.zip$`)),
	func(fileName, _ string) string { return strings.TrimSuffix(fileName, ".zip") + "/" },
)

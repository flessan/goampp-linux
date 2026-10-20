//go:build windows

// Package ruby registers Ruby (RubyInstaller) with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package ruby

import (
	"regexp"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Ruby", packages.DownloadSpec{

		Version:    "3.4.4",
		URL:        "https://github.com/oneclick/rubyinstaller2/releases/download/RubyInstaller-3.4.4-1/rubyinstaller-3.4.4-1-x64.exe",
		FileName:   "rubyinstaller-3.4.4-1-x64.exe",
		InstallDir: "bin/ruby",
		Kind:       packages.Exe,
		CheckFile:  "bin/ruby.exe",
		Notes:      "Ruby 3.4.4 via RubyInstaller2 — silent NSIS install.",
	}, packages.MapResolverVersion(
		packages.ResolveGitHubLatestAsset(
			"oneclick", "rubyinstaller2",
			regexp.MustCompile(`^rubyinstaller-[0-9.]+-[0-9]+-x64\.exe$`),
		),
		func(v string) string { return strings.TrimPrefix(v, "RubyInstaller-") },
	))
}

//go:build windows

// Package elixir registers Elixir with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package elixir

import (
	"regexp"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Elixir", packages.DownloadSpec{

		Version:    "1.18.3 (OTP 27)",
		URL:        "https://github.com/elixir-lang/elixir/releases/download/v1.18.3/elixir-otp-27.zip",
		FileName:   "elixir-otp-27.zip",
		InstallDir: "bin/elixir",
		StripTop:   "",
		Kind:       packages.Zip,
		CheckFile:  "bin/elixir.bat",
		Notes:      "Elixir 1.18.3 — requires Erlang OTP 27 (install Erlang card first).",
	}, packages.MapResolverVersion(
		packages.ResolveGitHubLatestAsset(
			"elixir-lang", "elixir",
			regexp.MustCompile(`^elixir-otp-27\.zip$`),
		),
		func(v string) string { return v + " (OTP 27)" },
	))
}

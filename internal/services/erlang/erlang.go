//go:build windows

// Package erlang registers Erlang/OTP with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package erlang

import (
	"regexp"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Erlang", packages.DownloadSpec{
		Version:    "27.3.4 (OTP-27)",
		URL:        "https://github.com/erlang/otp/releases/download/OTP-27.3.4/otp_win64_27.3.4.exe",
		FileName:   "otp_win64_27.3.4.exe",
		InstallDir: "bin/erlang",
		Kind:       packages.Exe,
		CheckFile:  "bin/erl.exe",
		Notes:      "Erlang/OTP runtime — required by RabbitMQ. Installed silently.",
	}, packages.MapResolverVersion(
		packages.ResolveGitHubSeriesAsset(
			"erlang", "otp", "OTP-27.",
			regexp.MustCompile(`^otp_win64_27\.[0-9.]+\.exe$`),
		),
		func(v string) string {
			return strings.TrimPrefix(v, "OTP-") + " (OTP-27)"
		},
	))
}

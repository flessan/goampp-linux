//go:build windows

// Package java registers Eclipse Temurin (Java) with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package java

import (
	"regexp"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Java", packages.DownloadSpec{
		Version:    "Temurin JDK 21.0.10+7",
		URL:        "https://github.com/adoptium/temurin21-binaries/releases/download/jdk-21.0.10%2B7/OpenJDK21U-jdk_x64_windows_hotspot_21.0.10_7.zip",
		FileName:   "OpenJDK21U-jdk_x64_windows_hotspot_21.0.10_7.zip",
		InstallDir: "bin/java",
		StripTop:   "jdk-21.0.10+7/",
		Kind:       packages.Zip,
		CheckFile:  "bin/java.exe",
		Notes:      "Eclipse Temurin LTS — javac, java, jar all under bin/.",
	}, resolveTemurin21Latest)
}

func resolveTemurin21Latest(log func(string)) (url, fileName, stripTop, version string, err error) {
	url, fileName, _, rawVersion, err := packages.ResolveGitHubLatestAsset(
		"adoptium", "temurin21-binaries",
		regexp.MustCompile(`^OpenJDK21U-jdk_x64_windows_hotspot_.*\.zip$`),
	)(log)
	if err != nil {
		return "", "", "", "", err
	}
	jdkVersion := strings.TrimPrefix(rawVersion, "jdk-")
	return url, fileName, "jdk-" + jdkVersion + "/", "Temurin JDK " + jdkVersion, nil
}

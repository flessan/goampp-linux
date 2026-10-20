//go:build windows

// Package haskell registers GHC (Haskell) with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package haskell

import (
	"fmt"
	"regexp"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Haskell", packages.DownloadSpec{

		Version:    "GHC 9.14.1",
		URL:        "https://downloads.haskell.org/ghc/9.14.1/ghc-9.14.1-x86_64-unknown-mingw32.zip",
		FileName:   "ghc-9.14.1-x86_64-unknown-mingw32.zip",
		InstallDir: "bin/haskell",
		StripTop:   "ghc-9.14.1/",
		Kind:       packages.Zip,
		CheckFile:  "bin/ghc.exe",
		Notes:      "GHC 9.14.1 — Glasgow Haskell Compiler. Large download (~570 MB).",
	}, resolveGHCLatest)
}

func resolveGHCLatest(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	const pageURL = "https://downloads.haskell.org/ghc/latest/"
	log("  resolving latest stable GHC Windows x86_64 archive ...")
	body, err := packages.FetchReleaseText(pageURL)
	if err != nil {
		return "", "", "", "", err
	}
	re := regexp.MustCompile(`(ghc-([0-9]+\.[0-9]+\.[0-9]+)-x86_64-unknown-mingw32\.zip)`)
	m := re.FindStringSubmatch(body)
	if len(m) < 3 {
		return "", "", "", "", fmt.Errorf("GHC Windows zip not found")
	}
	return pageURL + m[1], m[1], "ghc-" + m[2] + "/", "GHC " + m[2], nil
}

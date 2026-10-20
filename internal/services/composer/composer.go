//go:build windows

// Package composer registers Composer with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package composer

import (
	"encoding/json"
	"fmt"
	"strings"

	"goampp/internal/packages"
)

func init() {
	// Composer is its own tool in bin/composer. Keeping it in bin/php made PHP
	// updates and version switches capable of removing the PHAR or wrapper.
	packages.Register("Composer", packages.DownloadSpec{
		Version:     "latest stable",
		URL:         "https://getcomposer.org/composer-stable.phar",
		FileName:    "composer-stable.phar",
		InstallDir:  "bin/composer",
		Kind:        packages.File,
		TargetFile:  "composer.phar",
		CheckFile:   "composer.phar",
		Notes:       "Composer — PHP dependency manager, with a composer.bat wrapper.",
		PostInstall: packages.ComposerPostInstall,
	}, resolveComposerLatest)
}

const composerVersionsURL = "https://getcomposer.org/versions"

func composerVersionFileName(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "composer-stable.phar"
	}
	return "composer-" + version + ".phar"
}

func resolveComposerLatest(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	log("  resolving latest Composer stable release ...")
	body, err := packages.FetchReleaseText(composerVersionsURL)
	if err != nil {
		return "", "", "", "", err
	}

	var channels map[string][]struct {
		Path    string `json:"path"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(body), &channels); err != nil {
		return "", "", "", "", err
	}
	stable := channels["stable"]
	if len(stable) == 0 {
		return "", "", "", "", fmt.Errorf("Composer stable release missing")
	}

	best := stable[0]
	for _, candidate := range stable[1:] {
		if packages.NumericVersionGT(candidate.Version, best.Version) {
			best = candidate
		}
	}
	if best.Version == "" || best.Path == "" {
		return "", "", "", "", fmt.Errorf("Composer stable release metadata incomplete")
	}

	dlURL := best.Path
	if strings.HasPrefix(dlURL, "/") {
		dlURL = "https://getcomposer.org" + dlURL
	} else if !strings.HasPrefix(dlURL, "http://") && !strings.HasPrefix(dlURL, "https://") {
		dlURL = "https://getcomposer.org/" + strings.TrimLeft(dlURL, "/")
	}
	log("  latest Composer stable: " + best.Version)
	return dlURL, composerVersionFileName(best.Version), "", best.Version, nil
}

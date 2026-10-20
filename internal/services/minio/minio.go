//go:build windows

// Package minio registers MinIO with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package minio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("MinIO", packages.DownloadSpec{

		Version:    "latest",
		URL:        "https://dl.min.io/server/minio/release/windows-amd64/minio.exe",
		FileName:   "minio.exe",
		InstallDir: "bin/minio",
		Kind:       packages.File,
		TargetFile: "minio.exe",
		CheckFile:  "minio.exe",
		Notes:      "S3-compatible object storage — API :9010, console at http://localhost:9011 (user: minioadmin).",
		PostInstall: func(installDir string, log func(string)) error {
			baseDir := filepath.Dir(filepath.Dir(installDir))
			dataDir := filepath.Join(baseDir, "data", "minio")
			if err := os.MkdirAll(dataDir, 0o755); err != nil {
				return fmt.Errorf("create minio data dir: %w", err)
			}
			log("  created data/minio/ — MinIO will store objects here")
			return nil
		},
	}, resolveGitHubLatestTag(
		"minio", "minio",
		"https://dl.min.io/server/minio/release/windows-amd64/minio.exe",
		"minio.exe", "",
	))
}

func resolveGitHubLatestTag(owner, repo, downloadURL, fileName, stripTop string) packages.LatestResolver {
	return func(log func(string)) (url, file, strip, version string, err error) {
		log(fmt.Sprintf("  resolving latest %s/%s release ...", owner, repo))
		release, err := packages.GithubLatestRelease(owner, repo, log)
		if err != nil {
			return "", "", "", "", err
		}
		return downloadURL, fileName, stripTop, strings.TrimPrefix(release.TagName, "v"), nil
	}
}

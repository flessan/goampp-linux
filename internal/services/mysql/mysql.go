//go:build windows

// Package mysql registers MySQL (MariaDB) with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package mysql

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("MySQL", packages.DownloadSpec{

		Version:    "MariaDB 11.4.10 LTS",
		URL:        "https://archive.mariadb.org/mariadb-11.4.10/winx64-packages/mariadb-11.4.10-winx64.zip",
		FileName:   "mariadb-11.4.10-winx64.zip",
		InstallDir: "bin/mysql",
		StripTop:   "mariadb-11.4.10-winx64/",
		Kind:       packages.Zip,

		CheckFile: "data/mysql",
		Notes:     "MariaDB — MySQL-compatible drop-in.",
		PostInstall: func(installDir string, log func(string)) error {
			dataDir := filepath.Join(installDir, "data")
			if _, err := os.Stat(filepath.Join(dataDir, "mysql")); err == nil {
				log("  data dir already initialized, skipping")
				return nil
			}

			if entries, _ := os.ReadDir(dataDir); len(entries) > 0 {
				log("  wiping stale data/ from earlier broken install")
				if err := os.RemoveAll(dataDir); err != nil {
					return fmt.Errorf("wipe data dir: %w", err)
				}
				if err := os.MkdirAll(dataDir, 0o755); err != nil {
					return fmt.Errorf("recreate data dir: %w", err)
				}
			}

			candidates := []string{
				filepath.Join(installDir, "bin", "mariadb-install-db.exe"),
				filepath.Join(installDir, "bin", "mysql_install_db.exe"),
			}
			var initExe string
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					initExe = c
					break
				}
			}
			if initExe == "" {
				return fmt.Errorf("mariadb-install-db.exe not found")
			}
			log("  initializing MariaDB data directory (this can take ~30s)...")
			cmd := exec.Command(initExe, "--datadir="+dataDir)
			cmd.Dir = filepath.Join(installDir, "bin")
			out, err := cmd.CombinedOutput()
			if err != nil {

				log("  install-db output: " + strings.TrimSpace(string(out)))
				return err
			}
			return nil
		},
	}, resolveMariaDB114Latest)
}

type mariaDBLatestResponse struct {
	Releases map[string]struct {
		ReleaseID string `json:"release_id"`
		Files     []struct {
			FileName        string `json:"file_name"`
			FileDownloadURL string `json:"file_download_url"`
		} `json:"files"`
	} `json:"releases"`
}

func resolveMariaDB114Latest(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	const apiURL = "https://downloads.mariadb.org/rest-api/mariadb/11.4/latest/"
	log("  resolving latest MariaDB 11.4 LTS Windows build ...")

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", "", "", err
	}
	req.Header.Set("User-Agent", "GoAMPP")
	resp, err := packages.LatestHTTPClient.Do(req)
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", "", "", fmt.Errorf("MariaDB API returned HTTP %d", resp.StatusCode)
	}

	var data mariaDBLatestResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", "", "", err
	}
	assetRe := regexp.MustCompile(`^mariadb-([0-9]+\.[0-9]+\.[0-9]+)-winx64\.zip$`)
	for releaseKey, release := range data.Releases {
		for _, file := range release.Files {
			m := assetRe.FindStringSubmatch(file.FileName)
			if len(m) != 2 {
				continue
			}
			v := m[1]
			if release.ReleaseID != "" {
				v = release.ReleaseID
			} else if releaseKey != "" {
				v = releaseKey
			}
			if !strings.HasPrefix(v, "11.4.") {
				continue
			}
			log("  latest MariaDB 11.4 LTS: " + v)
			dlURL := strings.Replace(file.FileDownloadURL, "http://", "https://", 1)
			return dlURL, file.FileName, strings.TrimSuffix(file.FileName, ".zip") + "/", "MariaDB " + v + " LTS", nil
		}
	}
	return "", "", "", "", fmt.Errorf("MariaDB 11.4 winx64 zip not found")
}

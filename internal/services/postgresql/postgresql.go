//go:build windows

// Package postgresql registers PostgreSQL with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package postgresql

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("PostgreSQL", packages.DownloadSpec{

		Version:    "16.6 LTS (EDB)",
		URL:        "https://get.enterprisedb.com/postgresql/postgresql-16.6-1-windows-x64-binaries.zip",
		FileName:   "postgresql-16.6-1-windows-x64-binaries.zip",
		InstallDir: "bin/pgsql",
		StripTop:   "pgsql/",
		Kind:       packages.Zip,

		CheckFile: "data/PG_VERSION",
		Notes:     "EDB 16.6 LTS — most stable Postgres on Windows; supported until 2028.",
		PostInstall: func(installDir string, log func(string)) error {
			dataDir := filepath.Join(installDir, "data")
			if _, err := os.Stat(filepath.Join(dataDir, "PG_VERSION")); err == nil {
				log("  cluster already initialized, skipping")
				return nil
			}
			initExe := filepath.Join(installDir, "bin", "initdb.exe")
			if _, err := os.Stat(initExe); err != nil {
				return fmt.Errorf("initdb.exe not found")
			}

			pgBin := filepath.Join(installDir, "bin")
			runtimeDir := filepath.Dir(filepath.Dir(installDir))
			runtimeSrc := filepath.Join(runtimeDir, "runtime")
			if _, err := os.Stat(runtimeSrc); err == nil {
				for _, dll := range []string{"vcruntime140.dll", "vcruntime140_1.dll", "msvcp140.dll"} {
					_ = packages.CopyFile(
						filepath.Join(runtimeSrc, dll),
						filepath.Join(pgBin, dll),
					)
				}
				log("  seeded VC++ 14.44 runtime into pgsql/bin/")
			}

			pwTmp := filepath.Join(os.TempDir(), "goampp-pg-pw.txt")
			if err := os.WriteFile(pwTmp, []byte("postgres"), 0o600); err != nil {
				return fmt.Errorf("write pw file: %w", err)
			}
			defer os.Remove(pwTmp)

			log("  initializing PostgreSQL cluster (user=postgres, password=postgres, locale=C, auth=trust)...")
			cmd := exec.Command(initExe,
				"-D", dataDir,
				"-U", "postgres",
				"--pwfile="+pwTmp,
				"-E", "SQL_ASCII",
				"-A", "trust",
				"--locale=C",
				"--no-instructions",
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				log("  initdb output: " + strings.TrimSpace(string(out)))
				return err
			}
			return nil
		},
	}, resolvePostgreSQL16Latest)
}

// postgresLatestMinor reads the newest minor of a major release from the
// PostgreSQL project's own version index.
func postgresLatestMinor(major string) (string, error) {
	body, err := packages.FetchReleaseText("https://www.postgresql.org/versions.json")
	if err != nil {
		return "", err
	}
	var versions []struct {
		Major       string      `json:"major"`
		LatestMinor json.Number `json:"latestMinor"`
	}
	if err := json.Unmarshal([]byte(body), &versions); err != nil {
		return "", err
	}
	for _, v := range versions {
		if v.Major == major && v.LatestMinor != "" {
			return major + "." + v.LatestMinor.String(), nil
		}
	}
	return "", fmt.Errorf("PostgreSQL %s missing from versions.json", major)
}

func resolvePostgreSQL16Latest(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	log("  resolving latest PostgreSQL 16 Windows binary archive ...")
	v, err := postgresLatestMinor("16")
	if err != nil {
		// Fall back to the EDB page. Its markup splits "Version <!-- -->16.x".
		body, perr := packages.FetchReleaseText("https://www.enterprisedb.com/download-postgresql-binaries")
		if perr != nil {
			return "", "", "", "", fmt.Errorf("%v; %v", err, perr)
		}
		m := packages.EdbPostgres16Re.FindStringSubmatch(body)
		if len(m) < 2 {
			return "", "", "", "", fmt.Errorf("PostgreSQL 16 binary version not found")
		}
		v = m[1]
	}

	// EDB occasionally republishes an installer point release with a new build
	// suffix. Probe a few build numbers without downloading the archive.
	for build := 4; build >= 1; build-- {
		file := fmt.Sprintf("postgresql-%s-%d-windows-x64-binaries.zip", v, build)
		dlURL := "https://get.enterprisedb.com/postgresql/" + file
		req, err := http.NewRequest(http.MethodHead, dlURL, nil)
		if err != nil {
			continue
		}
		resp, err := packages.LatestHTTPClient.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			log(fmt.Sprintf("  latest PostgreSQL 16 binary: %s (EDB build %d)", v, build))
			return dlURL, file, "pgsql/", v, nil
		}
	}
	return "", "", "", "", fmt.Errorf("no reachable EDB PostgreSQL %s x64 binary archive", v)
}

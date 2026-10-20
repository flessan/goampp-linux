//go:build windows

// Package adminer registers Adminer with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package adminer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Adminer", packages.DownloadSpec{
		Version:    "5.4.2",
		URL:        "https://github.com/vrana/adminer/releases/download/v5.4.2/adminer-5.4.2-en.php",
		FileName:   "adminer-5.4.2-en.php",
		InstallDir: "www/adminer",
		Kind:       packages.File,
		TargetFile: "index.php",
		CheckFile:  "index.php",
		PostInstall: func(installDir string, log func(string)) error {

			target := filepath.Join(installDir, "index.php")
			data, err := os.ReadFile(target)
			if err != nil {
				return fmt.Errorf("read adminer: %w", err)
			}
			re := regexp.MustCompile(`if\(\$\w+==""\)return\s+sprintf\('Adminer does not support`)
			patched := re.ReplaceAllLiteralString(string(data),
				`if(false)return sprintf('Adminer does not support`)
			if patched == string(data) {
				log("  Adminer: empty-password guard pattern not found (already patched?)")
				return nil
			}
			if err := os.WriteFile(target, []byte(patched), 0o644); err != nil {
				return fmt.Errorf("write patched adminer: %w", err)
			}
			log("  patched Adminer to allow blank-password local dev logins")
			return nil
		},
	}, resolveAdminerLatest)
}

var resolveAdminerLatest = packages.ResolveGitHubLatestAsset(
	"vrana",
	"adminer",
	regexp.MustCompile(`^adminer-[0-9.]+-en\.php$`),
)

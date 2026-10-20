//go:build windows

// Package php registers PHP (CGI, with selectable branches) with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package php

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("PHP-FPM", packages.DownloadSpec{

		Version:    "8.4.22 NTS x64",
		URL:        "https://downloads.php.net/~windows/releases/archives/php-8.4.22-nts-Win32-vs17-x64.zip",
		FileName:   "php-8.4.22-nts-Win32-vs17-x64.zip",
		InstallDir: "bin/php",

		Variants: []packages.VariantSpec{
			{Version: "7.4", URL: "https://windows.php.net/downloads/releases/archives/php-7.4.33-nts-Win32-vc15-x64.zip", FileName: "php-7.4.33-nts-Win32-vc15-x64.zip", Notes: "PHP 7.4 — needs VC15 (VS2017) runtime."},
			{Version: "8.0", URL: "https://windows.php.net/downloads/releases/archives/php-8.0.30-nts-Win32-vs16-x64.zip", FileName: "php-8.0.30-nts-Win32-vs16-x64.zip", Notes: "PHP 8.0 — VS16 (VS2019) runtime."},
			{Version: "8.1", URL: "https://windows.php.net/downloads/releases/archives/php-8.1.31-nts-Win32-vs16-x64.zip", FileName: "php-8.1.31-nts-Win32-vs16-x64.zip", Notes: "PHP 8.1 — VS16 (VS2019) runtime."},
			{Version: "8.2", URL: "https://windows.php.net/downloads/releases/archives/php-8.2.27-nts-Win32-vs16-x64.zip", FileName: "php-8.2.27-nts-Win32-vs16-x64.zip", Notes: "PHP 8.2 — VS16 (VS2019) runtime."},
			{Version: "8.3", URL: "https://windows.php.net/downloads/releases/archives/php-8.3.15-nts-Win32-vs16-x64.zip", FileName: "php-8.3.15-nts-Win32-vs16-x64.zip", Notes: "PHP 8.3 — VS16 (VS2019) runtime."},
			{Version: "8.4", URL: "https://downloads.php.net/~windows/releases/archives/php-8.4.22-nts-Win32-vs17-x64.zip", FileName: "php-8.4.22-nts-Win32-vs17-x64.zip", Notes: "PHP 8.4 — VS17 (VS2022) runtime, v14.4+."},
			{Version: "8.5", URL: "https://downloads.php.net/~windows/releases/archives/php-8.5.7-nts-Win32-vs17-x64.zip", FileName: "php-8.5.7-nts-Win32-vs17-x64.zip", Notes: "PHP 8.5 — VS17 (VS2022) runtime, v14.4+."},
		},
		StripTop:  "",
		Kind:      packages.Zip,
		CheckFile: "php-cgi.exe",
		Notes:     "Requires VC++ 2015-2022 Redistributable x64.",
		PostInstall: func(installDir string, log func(string)) error {

			iniPath := filepath.Join(installDir, "php.ini")
			var data []byte
			if b, err := os.ReadFile(iniPath); err == nil {
				data = b
			} else {
				dev := filepath.Join(installDir, "php.ini-development")
				b, derr := os.ReadFile(dev)
				if derr != nil {
					return nil
				}
				data = b
				log("  created php.ini from php.ini-development")
			}

			baseDir := filepath.Dir(filepath.Dir(installDir))
			tmpDir := strings.ReplaceAll(filepath.Join(baseDir, "tmp"), `\`, `/`)
			if err := os.MkdirAll(tmpDir, 0o755); err != nil {
				return err
			}

			s := string(data)

			patches := []struct{ re, replacement string }{
				{`(?m)^;?\s*extension_dir\s*=\s*"\./"`, `;extension_dir = "./"`},
				{`(?m)^;?\s*extension_dir\s*=\s*"ext"`, `extension_dir = "ext"`},
				{`(?m)^;?\s*upload_tmp_dir\s*=.*`, `upload_tmp_dir = "` + tmpDir + `"`},
				{`(?m)^;?\s*session\.save_path\s*=.*`, `session.save_path = "` + tmpDir + `"`},
				{`(?m)^;?\s*cgi\.force_redirect\s*=.*`, `cgi.force_redirect = 0`},
				{`(?m)^;?\s*cgi\.fix_pathinfo\s*=.*`, `cgi.fix_pathinfo=1`},

				{`(?m)^;?\s*display_errors\s*=.*`, `display_errors = Off`},
				{`(?m)^;?\s*display_startup_errors\s*=.*`, `display_startup_errors = Off`},
			}
			for _, p := range patches {
				s = regexp.MustCompile(p.re).ReplaceAllString(s, p.replacement)
			}

			exts := []string{

				"bz2", "curl", "exif", "fileinfo", "ftp", "gd",
				"gettext", "gmp", "mbstring", "openssl", "zip",

				"intl",

				"mysqli", "pdo_mysql",
				"pdo_pgsql", "pgsql",
				"pdo_sqlite", "sqlite3",

				"soap", "sockets", "xsl",

				"sodium",
			}
			for _, e := range exts {
				re := regexp.MustCompile(`(?m)^;extension\s*=\s*` + e + `\b.*`)
				s = re.ReplaceAllString(s, "extension="+e)
			}

			if err := os.WriteFile(iniPath, []byte(s), 0o644); err != nil {
				return err
			}
			log("  patched php.ini (extension_dir, session.save_path, 8 extensions)")

			packages.EnsurePhpRuntimeDLLs(baseDir, log)
			return nil
		},
	}, nil)
}

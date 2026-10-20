//go:build windows

// Package python registers Python (embeddable) with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package python

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Python", packages.DownloadSpec{
		Version:    "3.13.13 (embeddable)",
		URL:        "https://www.python.org/ftp/python/3.13.13/python-3.13.13-embed-amd64.zip",
		FileName:   "python-3.13.13-embed-amd64.zip",
		InstallDir: "bin/python",
		StripTop:   "",
		Kind:       packages.Zip,
		CheckFile:  "python.exe",
		Notes:      "Embeddable Python — pip is bootstrapped automatically post-install.",
		Variants: []packages.VariantSpec{
			{Version: "3.10", URL: "https://www.python.org/ftp/python/3.10.11/python-3.10.11-embed-amd64.zip", FileName: "python-3.10.11-embed-amd64.zip", Notes: "Python 3.10 (security-only)."},
			{Version: "3.11", URL: "https://www.python.org/ftp/python/3.11.9/python-3.11.9-embed-amd64.zip", FileName: "python-3.11.9-embed-amd64.zip", Notes: "Python 3.11."},
			{Version: "3.12", URL: "https://www.python.org/ftp/python/3.12.7/python-3.12.7-embed-amd64.zip", FileName: "python-3.12.7-embed-amd64.zip", Notes: "Python 3.12."},
			{Version: "3.13", URL: "https://www.python.org/ftp/python/3.13.13/python-3.13.13-embed-amd64.zip", FileName: "python-3.13.13-embed-amd64.zip", Notes: "Python 3.13 — current default."},
		},
		PostInstall: func(installDir string, log func(string)) error {

			files, _ := filepath.Glob(filepath.Join(installDir, "python*._pth"))
			for _, p := range files {
				data, err := os.ReadFile(p)
				if err == nil {
					patched := strings.ReplaceAll(string(data),
						"#import site", "import site")
					if patched != string(data) {
						log("  uncommented import site in " + filepath.Base(p))
						_ = os.WriteFile(p, []byte(patched), 0o644)
					}
				}
			}

			pipExe := filepath.Join(installDir, "Scripts", "pip.exe")
			if _, err := os.Stat(pipExe); err == nil {
				log("  pip already installed")
				return nil
			}
			getPip := filepath.Join(installDir, "get-pip.py")
			log("  downloading get-pip.py from bootstrap.pypa.io")
			if err := packages.HttpDownload("https://bootstrap.pypa.io/get-pip.py", getPip, log, nil); err != nil {
				log("  get-pip download failed (non-fatal): " + err.Error())
				return nil
			}
			log("  bootstrapping pip via python.exe get-pip.py ...")
			pyExe := filepath.Join(installDir, "python.exe")
			cmd := exec.Command(pyExe, getPip, "--no-warn-script-location", "--quiet")
			cmd.Dir = installDir
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
			if out, err := cmd.CombinedOutput(); err != nil {
				log("  pip bootstrap output: " + strings.TrimSpace(string(out)))
				return err
			}
			log("  pip ready")
			return nil
		},
	}, nil)
}

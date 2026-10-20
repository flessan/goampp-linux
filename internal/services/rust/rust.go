//go:build windows

// Package rust registers Rust (rustup) with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package rust

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Rust", packages.DownloadSpec{

		Version:    "stable (via rustup)",
		URL:        "https://static.rust-lang.org/rustup/dist/x86_64-pc-windows-msvc/rustup-init.exe",
		FileName:   "rustup-init.exe",
		InstallDir: "bin/rust",
		Kind:       packages.File,
		TargetFile: "rustup-init.exe",
		CheckFile:  ".cargo/bin/cargo.exe",
		Notes:      "Rust stable toolchain — cargo + rustc land in bin/rust/.cargo/bin/.",
		PostInstall: func(installDir string, log func(string)) error {
			rustupInit := filepath.Join(installDir, "rustup-init.exe")
			if _, err := os.Stat(rustupInit); err != nil {
				return fmt.Errorf("rustup-init.exe not found in %s", installDir)
			}
			log("  installing Rust stable via rustup (3–5 min first time)...")
			cmd := exec.Command(rustupInit,
				"-y",
				"--no-modify-path",
				"--default-toolchain", "stable",
				"--profile", "default",
			)
			cmd.Env = append(os.Environ(),
				"RUSTUP_HOME="+filepath.Join(installDir, ".rustup"),
				"CARGO_HOME="+filepath.Join(installDir, ".cargo"),
			)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
			if out, err := cmd.CombinedOutput(); err != nil {
				log("  rustup output: " + strings.TrimSpace(string(out)))
				return err
			}
			log("  Rust stable ready — cargo at bin/rust/.cargo/bin/cargo.exe")
			return nil
		},
	}, resolveRustStable)
}

func resolveRustStable(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	const manifestURL = "https://static.rust-lang.org/dist/channel-rust-stable.toml"
	log("  resolving latest Rust stable toolchain ...")
	body, err := packages.FetchReleaseText(manifestURL)
	if err != nil {
		return "", "", "", "", err
	}
	re := regexp.MustCompile(`(?s)\[pkg\.rust\].*?version\s*=\s*"([0-9]+\.[0-9]+\.[0-9]+)`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return "", "", "", "", fmt.Errorf("Rust stable version not found")
	}
	return "https://static.rust-lang.org/rustup/dist/x86_64-pc-windows-msvc/rustup-init.exe",
		"rustup-init.exe", "", m[1] + " stable", nil
}

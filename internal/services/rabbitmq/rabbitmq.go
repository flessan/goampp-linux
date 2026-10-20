//go:build windows

// Package rabbitmq registers RabbitMQ with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package rabbitmq

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
	packages.Register("RabbitMQ", packages.DownloadSpec{
		Version:    "4.3.0",
		URL:        "https://github.com/rabbitmq/rabbitmq-server/releases/download/v4.3.0/rabbitmq-server-windows-4.3.0.zip",
		FileName:   "rabbitmq-server-windows-4.3.0.zip",
		InstallDir: "bin/rabbitmq",
		StripTop:   "rabbitmq_server-4.3.0/",
		Kind:       packages.Zip,
		CheckFile:  "sbin/rabbitmq-server.bat",
		Notes:      "AMQP message broker — AMQP :5672, management UI :15672 (guest/guest).",
		PostInstall: func(installDir string, log func(string)) error {
			baseDir := filepath.Dir(filepath.Dir(installDir))

			erlangDir := filepath.Join(baseDir, "bin", "erlang")
			if _, err := os.Stat(filepath.Join(erlangDir, "bin", "erl.exe")); err != nil {
				log("  Erlang not found — downloading OTP 27.3.4...")
				const erlURL = "https://github.com/erlang/otp/releases/download/OTP-27.3.4/otp_win64_27.3.4.exe"
				erlExe, err := packages.CacheFor(baseDir, log).Fetch("otp_win64_27.3.4.exe", erlURL, nil)
				if err != nil {
					return fmt.Errorf("erlang download: %w", err)
				}
				log("  installing Erlang OTP silently...")
				if err := os.MkdirAll(erlangDir, 0o755); err != nil {
					return fmt.Errorf("erlang dir: %w", err)
				}
				abs, _ := filepath.Abs(erlangDir)
				cmd := exec.Command(erlExe, "/S", "/D="+abs)
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
				if out, err := cmd.CombinedOutput(); err != nil {
					log("  erlang installer: " + strings.TrimSpace(string(out)))
					return fmt.Errorf("erlang install: %w", err)
				}
				log("  Erlang OTP installed")
			}

			dataDir := filepath.Join(baseDir, "data", "rabbitmq")
			if err := os.MkdirAll(dataDir, 0o755); err != nil {
				return fmt.Errorf("create rabbitmq data dir: %w", err)
			}
			log("  created data/rabbitmq/ — RabbitMQ stores state here")

			pluginsExe := filepath.Join(installDir, "sbin", "rabbitmq-plugins.bat")
			if _, err := os.Stat(pluginsExe); err == nil {
				cmd := exec.Command("cmd.exe", "/c", pluginsExe, "enable", "rabbitmq_management")
				cmd.Env = append(os.Environ(),
					"ERLANG_HOME="+erlangDir,
					"RABBITMQ_BASE="+dataDir,
				)
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
				if out, err := cmd.CombinedOutput(); err != nil {
					log("  rabbitmq-plugins: " + strings.TrimSpace(string(out)))

				} else {
					log("  enabled rabbitmq_management plugin")
				}
			}
			return nil
		},
	}, packages.WithStripTop(
		packages.ResolveGitHubSeriesAsset(
			"rabbitmq", "rabbitmq-server", "v4.3.",
			regexp.MustCompile(`^rabbitmq-server-windows-4\.3\.[0-9]+\.zip$`),
		),
		func(_, version string) string {
			return "rabbitmq_server-" + strings.TrimPrefix(version, "v") + "/"
		},
	))

	// The post-install above still carries an OTP 27.3.4 fallback. Install the
	// dynamically resolved OTP 27 first; the fallback then finds erl.exe and
	// skips its hard-coded download.
	spec := packages.DownloadCatalog["RabbitMQ"]
	originalPostInstall := spec.PostInstall
	spec.PostInstall = func(installDir string, log func(string)) error {
		if err := ensureRabbitMQErlang(installDir, log); err != nil {
			return err
		}
		if originalPostInstall != nil {
			return originalPostInstall(installDir, log)
		}
		return nil
	}
	packages.DownloadCatalog["RabbitMQ"] = spec
}

func ensureRabbitMQErlang(installDir string, log func(string)) error {
	baseDir := filepath.Dir(filepath.Dir(installDir))
	erlangDir := filepath.Join(baseDir, "bin", "erlang")
	if _, err := os.Stat(filepath.Join(erlangDir, "bin", "erl.exe")); err == nil {
		return nil
	}

	candidates, err := packages.DownloadCandidates("Erlang", "", log)
	if err != nil {
		return fmt.Errorf("resolve Erlang dependency: %w", err)
	}
	log("  Erlang not found — downloading OTP...")
	erlExe, used, err := packages.FetchFirstCandidate(packages.CacheFor(baseDir, log), "Erlang", candidates, log, nil)
	if err != nil {
		return fmt.Errorf("erlang download: %w", err)
	}
	resolved := packages.DownloadResolution{
		Version:   used.Version,
		PackageID: packages.PackageIdentity(used.Version, used.FileName),
		URL:       used.URL,
		FileName:  used.FileName,
		StripTop:  used.StripTop,
	}

	log("  installing Erlang OTP silently...")
	if err := os.MkdirAll(erlangDir, 0o755); err != nil {
		return fmt.Errorf("erlang dir: %w", err)
	}
	abs, _ := filepath.Abs(erlangDir)
	cmd := exec.Command(erlExe, "/S", "/D="+abs)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if out, err := cmd.CombinedOutput(); err != nil {
		log("  erlang installer: " + strings.TrimSpace(string(out)))
		return fmt.Errorf("erlang install: %w", err)
	}
	log("  Erlang OTP installed")

	// resolveLatestDownload marks resolver-backed installs for the normal
	// PostInstall metadata hook. RabbitMQ installs Erlang directly while the
	// download mutex is already held, so record the metadata here instead.
	packages.InstallResolveMu.Lock()
	delete(packages.ResolvedFreshInstall, "Erlang")
	packages.InstallResolveMu.Unlock()
	if err := packages.WriteInstalledPackage(baseDir, "Erlang", "", resolved); err != nil {
		log(fmt.Sprintf("  Erlang package metadata write failed: %v", err))
	} else {
		log(fmt.Sprintf("  recorded installed Erlang package %s", resolved.FileName))
	}
	return nil
}

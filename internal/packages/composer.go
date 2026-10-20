//go:build windows

package packages

import (
	"fmt"
	"os"
	"path/filepath"

	"goampp/internal/env"
	"goampp/internal/pathenv"
)

func composerDir(baseDir string) string {
	return filepath.Join(baseDir, "bin", "composer")
}

func ComposerPharPath(baseDir string) string {
	return filepath.Join(composerDir(baseDir), "composer.phar")
}

func legacyComposerPharPath(baseDir string) string {
	return filepath.Join(baseDir, "bin", "php", "composer.phar")
}

func ComposerPostInstall(installDir string, log func(string)) error {
	baseDir := filepath.Dir(filepath.Dir(installDir))
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	wrapper := "@echo off\r\n" +
		"setlocal\r\n" +
		"set \"GOAMPP_PHP=%~dp0..\\php\\php.exe\"\r\n" +
		"if not exist \"%GOAMPP_PHP%\" (\r\n" +
		"  echo GoAMPP PHP was not found at \"%GOAMPP_PHP%\". 1>&2\r\n" +
		"  exit /b 1\r\n" +
		")\r\n" +
		"\"%GOAMPP_PHP%\" \"%~dp0composer.phar\" %*\r\n"
	if err := os.WriteFile(filepath.Join(installDir, "composer.bat"), []byte(wrapper), 0o644); err != nil {
		return fmt.Errorf("write composer.bat: %w", err)
	}

	// Keep a tiny compatibility shim in bin/php for existing PATH entries.
	phpDir := filepath.Join(baseDir, "bin", "php")
	if fi, err := os.Stat(phpDir); err == nil && fi.IsDir() {
		shim := "@echo off\r\n" +
			"call \"%~dp0..\\composer\\composer.bat\" %*\r\n" +
			"exit /b %errorlevel%\r\n"
		if err := os.WriteFile(filepath.Join(phpDir, "composer.bat"), []byte(shim), 0o644); err != nil {
			return fmt.Errorf("write Composer compatibility shim: %w", err)
		}
	}

	if log != nil {
		log("  configured Composer wrapper")
	}
	return nil
}

// migrateLegacyComposer moves Composer out of bin/php so PHP upgrades and
// version switches cannot remove it. Existing PATH entries keep working via a
// small compatibility shim in bin/php.
func MigrateLegacyComposer(baseDir string, log func(string)) error {
	targetDir := composerDir(baseDir)
	target := ComposerPharPath(baseDir)
	legacy := legacyComposerPharPath(baseDir)

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	source := ""
	migrated := false
	if _, err := os.Stat(target); err != nil {
		switch {
		case composerFileExists(legacy):
			source = legacy
			migrated = true
		case composerFileExists(filepath.Join(baseDir, "downloads", "composer-stable.phar")):
			source = filepath.Join(baseDir, "downloads", "composer-stable.phar")
			migrated = true
		}
		if source != "" {
			if err := CopyFile(source, target); err != nil {
				return fmt.Errorf("migrate Composer: %w", err)
			}
		}
	}

	if !composerFileExists(target) {
		return nil
	}
	if err := ComposerPostInstall(targetDir, log); err != nil {
		return err
	}

	if composerFileExists(legacy) {
		if err := os.Remove(legacy); err != nil {
			return fmt.Errorf("remove legacy Composer PHAR: %w", err)
		}
	}
	if migrated && log != nil {
		log("composer: moved managed files from bin/php to bin/composer")
	}

	// Make the standalone Composer directory available to new terminals. The
	// compatibility shim above keeps existing bin/php PATH entries working too.
	if env.BaseDir != "" {
		if n, err := pathenv.AddGoamppToUserPath(); err != nil {
			if log != nil {
				log("composer: PATH update skipped: " + err.Error())
			}
		} else if n > 0 && log != nil {
			log(fmt.Sprintf("composer: added %d GoAMPP bin dir(s) to user PATH", n))
		}
	}
	return nil
}

func composerFileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

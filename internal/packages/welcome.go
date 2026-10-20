//go:build windows

package packages

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func EnsurePhpRuntimeDLLs(baseDir string, log func(string)) {
	runtimeDir := filepath.Join(baseDir, "runtime")
	if _, err := os.Stat(runtimeDir); err != nil {
		return
	}
	dlls := []string{"vcruntime140.dll", "vcruntime140_1.dll", "msvcp140.dll"}

	binDir := filepath.Join(baseDir, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name != "php" && !strings.HasPrefix(name, "php-") {
			continue
		}

		if strings.HasSuffix(name, "-legacy") {
			continue
		}
		phpDir := filepath.Join(binDir, name)
		if _, err := os.Stat(filepath.Join(phpDir, "php-cgi.exe")); err != nil {
			continue
		}
		for _, dll := range dlls {
			src := filepath.Join(runtimeDir, dll)
			dst := filepath.Join(phpDir, dll)
			if err := copyFileIfChanged(src, dst); err != nil {
				log("  self-heal: failed to mirror " + dll + " into " + name + ": " + err.Error())
			}
		}
	}
}

func copyFileIfChanged(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if dstInfo, err := os.Stat(dst); err == nil {
		if dstInfo.Size() == srcInfo.Size() && dstInfo.ModTime().Equal(srcInfo.ModTime()) {
			return nil
		}
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	return os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime())
}

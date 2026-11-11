//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestWindowsNativeBuild verifies the actual application executable, not only
// the individual packages compiled by go test, on the Windows CI runner.
func TestWindowsNativeBuild(t *testing.T) {
	output := filepath.Join(t.TempDir(), "goampp.exe")
	cmd := exec.Command("go", "build", "-trimpath", "-o", output, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Windows build failed: %v\n%s", err, out)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("Windows build did not produce an executable: %v", err)
	}
}

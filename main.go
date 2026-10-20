//go:build windows

// GoAMPP is a native Windows control panel for a local web stack.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"

	"goampp/internal/core"
	"goampp/internal/gui"
	_ "goampp/internal/services"
	"goampp/internal/vhost"
)

func main() {
	if vhost.HandleElevatedHostsHelper() {
		return
	}
	if hideRun() {
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		die("executable path: " + err.Error())
	}
	if err := core.Init(filepath.Dir(exePath)); err != nil {
		die(err.Error())
	}
	gui.Run(core.HasTrayFlag(), core.AutoStartServices)
}

// hideRun handles "goampp.exe --hide-run <exe> [args]": start a program with
// no console window and exit. Services use it for helpers that would
// otherwise flash a console.
func hideRun() bool {
	if len(os.Args) < 3 || os.Args[1] != "--hide-run" {
		return false
	}
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	_ = cmd.Start()
	return true
}

// die reports a fatal startup error. It runs before the GUI exists, so it
// uses a plain Win32 message box.
func die(msg string) {
	text, _ := windows.UTF16PtrFromString(msg)
	title, _ := windows.UTF16PtrFromString("GoAMPP — fatal error")
	_, _ = windows.MessageBox(0, text, title, windows.MB_ICONERROR|windows.MB_OK)
	os.Exit(1)
}

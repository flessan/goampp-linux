//go:build windows

// Package core is the application layer: the running state, every action a
// user can trigger (start, stop, install, switch versions, vhosts, projects)
// and update checks. Front ends drive it and receive callbacks via UIHooks.
package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"fyne.io/fyne/v2"

	"goampp/internal/config"
	"goampp/internal/env"
	"goampp/internal/packages"
	"goampp/internal/process"
	"goampp/internal/services/apache"
)

type State struct {
	BaseDir  string
	Cfg      *config.Config
	Services []*ManagedService

	Fyne fyne.App
	Win  fyne.Window

	LogMu    sync.Mutex
	logLines []string
	// onLog is set by the GUI once it exists. It is called from whatever
	// goroutine logged the line, so it must not touch widgets directly.
	OnLog func()
}

type ManagedService struct {
	Conf    *config.ServiceConf
	Service *process.Service
}

var App *State

const maxLogLines = 3000

// appVersion is the release this build came from. The release workflow stamps
// it from the tag (-ldflags "-X main.appVersion=v0.6.4"), so the About box and
// the status bar cannot drift from the release they were built from — which is
// what the two hand-written copies of this string used to do. "dev" is the
// honest label for a local build.
var Version = "dev"

// initApp loads the configuration and prepares everything that must be in
// place before a front end attaches.
func Init(baseDir string) error {
	cfg, err := config.LoadConfig(baseDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	App = &State{BaseDir: baseDir, Cfg: cfg}
	env.BaseDir = baseDir
	env.LogSink = App.AppendLog

	if err := packages.MigrateLegacyComposer(baseDir, App.AppendLog); err != nil {
		App.AppendLog("composer migration: " + err.Error())
	}

	// Before anything can reach for a cached download: drop files an earlier
	// run left poisoned, so an install that already hit the Apache Lounge
	// bug heals on this launch instead of failing at extract again (issue #1).
	packages.SetDownloadCache(baseDir, App.AppendLog)
	packages.SweepDownloadCache(baseDir, App.AppendLog)

	App.Services = BuildManagedServices(cfg, nil)

	process.KillZombieChildren(baseDir)

	App.AppendLog(fmt.Sprintf("GoAMPP %s started at %s", Version, time.Now().Format("15:04:05")))
	App.AppendLog(fmt.Sprintf("Base dir: %s", baseDir))
	App.AppendLog(fmt.Sprintf("Loaded %d services, %d vhosts from config.json",
		len(cfg.Services), len(cfg.Vhosts)))
	return nil
}

// autoStartServices repairs the Apache runtime files and starts the services
// ticked on the Startup page. It runs once the front end is up.
func AutoStartServices() {
	apachePath := filepath.Join(App.BaseDir, "bin", "apache", "bin", "httpd.exe")
	if _, err := os.Stat(apachePath); err == nil {
		apache.EnsureApacheRuntimeFiles(App.BaseDir, App.AppendLog)
	}

	for _, name := range App.Cfg.Settings.AutoStart {
		ms := App.findService(name)
		if !StartupServiceEligible(ms) {
			App.AppendLog(fmt.Sprintf("[auto-start] skipping unavailable service %s", name))
			continue
		}
		if err := ms.Service.Start(); err != nil {
			App.AppendLog(fmt.Sprintf("[auto-start] %s: %v", name, err))
		}
	}
}

// buildManagedServices wraps each configured service, reusing the live
// *Service from old so a config reload does not orphan running processes.
func BuildManagedServices(cfg *config.Config, old []*ManagedService) []*ManagedService {
	out := make([]*ManagedService, 0, len(cfg.Services))
	for i := range cfg.Services {
		sc := &cfg.Services[i]
		ms := &ManagedService{Conf: sc}
		for _, o := range old {
			if o.Conf.Name == sc.Name && o.Service != nil {
				ms.Service = o.Service
				break
			}
		}
		if ms.Service == nil && sc.ExePath != "" {
			ms.Service = &process.Service{
				Name:    sc.Name,
				ExePath: config.ExpandPath(sc.ExePath, App.BaseDir),
				Args:    expandArgsList(sc.Args, App.BaseDir),
				Port:    sc.Port,
				WorkDir: config.ExpandPath(sc.WorkDir, App.BaseDir),
				Env:     expandArgsList(sc.Env, App.BaseDir),
			}
			ms.Service.SetLogger(App.AppendLog)
			ms.Service.SetStateCallback(func(running bool, pid int) {
				UiRefreshServices()
			})
		}
		out = append(out, ms)
	}
	return out
}

func (a *State) findService(name string) *ManagedService {
	for _, ms := range a.Services {
		if ms.Conf.Name == name {
			return ms
		}
	}
	return nil
}

func (a *State) AppendLog(line string) {
	entry := time.Now().Format("15:04:05") + "  " + strings.TrimRight(line, "\r\n")

	a.LogMu.Lock()
	a.logLines = append(a.logLines, entry)
	if len(a.logLines) > maxLogLines {
		// Drop a quarter at once so trimming is not paid on every line.
		a.logLines = append([]string(nil), a.logLines[maxLogLines/4:]...)
	}
	notify := a.OnLog
	a.LogMu.Unlock()

	if notify != nil {
		notify()
	}
}

// logSnapshot returns a copy of the log lines that is safe to hand to widgets.
func (a *State) LogSnapshot() []string {
	a.LogMu.Lock()
	defer a.LogMu.Unlock()
	return append([]string(nil), a.logLines...)
}

func (a *State) ClearLog() {
	a.LogMu.Lock()
	a.logLines = nil
	a.LogMu.Unlock()
}

func expandArgsList(args []string, baseDir string) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = config.ExpandPath(a, baseDir)
	}
	return out
}

func OpenPath(path string) {
	cmd := exec.Command("cmd", "/c", "start", "", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
}

func OpenFolder(path string) {
	if _, err := os.Stat(path); err != nil {
		_ = os.MkdirAll(path, 0o755)
	}
	_ = exec.Command("explorer.exe", filepath.FromSlash(path)).Start()
}

func TruncateMid(s string, max int) string {
	if len(s) <= max {
		return s
	}
	keep := (max - 3) / 2
	return s[:keep] + "..." + s[len(s)-keep:]
}

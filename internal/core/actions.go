//go:build windows

package core

// GUI-independent actions: everything a button, the tray menu or auto-start
// can trigger. Widgets call these and re-render through refresh* hooks.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"

	"goampp/internal/config"
	"goampp/internal/packages"
	"goampp/internal/process"
	"goampp/internal/projects"
	"goampp/internal/vhost"
)

var DomainExtensions = []string{
	".test",
	".local",
	".localhost",
	".lan",
	".home",
	".site",
}

var essentialServices = []string{"Apache", "PHP-FPM", "MySQL", "phpMyAdmin"}

func ActiveWebServer() string {
	if App.Cfg != nil && App.Cfg.Settings.ActiveWebServer != "" {
		return App.Cfg.Settings.ActiveWebServer
	}
	return "Apache"
}

func essentialServicesActive() []string {
	out := make([]string, 0, len(essentialServices))
	web := ActiveWebServer()
	for _, name := range essentialServices {
		if name == "Apache" || name == "Nginx" {
			out = append(out, web)
			continue
		}
		out = append(out, name)
	}
	return out
}

// setActiveWebServer persists the choice, stops whichever other web server is
// holding the port and, when one was serving, brings the new one up in its
// place (installing it first if needed) with the vhost configs regenerated.
func SetActiveWebServer(choice string) {
	if choice == ActiveWebServer() {
		return
	}
	if App.Cfg != nil {
		App.Cfg.Settings.ActiveWebServer = choice
		_ = config.SaveConfig(App.BaseDir, App.Cfg)
	}
	App.AppendLog("active web server: " + choice)
	go func() {
		wasServing := false
		for _, ms := range App.Services {
			if !IsWebServerName(ms.Conf.Name) || ms.Conf.Name == choice {
				continue
			}
			if ms.Service != nil && ms.Service.Running() {
				wasServing = true
				App.AppendLog(fmt.Sprintf("[%s] stopped — switching active web server to %s", ms.Conf.Name, choice))
				_ = ms.Service.Stop()
			}
		}
		UiRefreshServices()
		if !wasServing {
			return
		}
		if len(App.Cfg.Vhosts) > 0 {
			if err := vhost.ApplyVhosts(App.BaseDir, App.Cfg); err != nil {
				App.AppendLog("apply vhosts: " + err.Error())
			}
		}
		StartService(App.findService(choice))
	}()
}

// restartWebServerIfRunning reloads the active web server so it picks up new
// vhosts or a different PHP build.
func RestartWebServerIfRunning() {
	ms := App.findService(ActiveWebServer())
	if ms != nil && ms.Service != nil && ms.Service.Running() {
		App.AppendLog(fmt.Sprintf("[%s] restarting to apply changes", ms.Conf.Name))
		RestartService(ms)
	}
}

// applyVhostsAuto writes hosts + web server configs and reloads the server.
// Every vhost/project edit goes through it, so there is no separate apply step
// to forget.
func ApplyVhostsAuto() {
	go func() {
		if err := vhost.ApplyVhosts(App.BaseDir, App.Cfg); err != nil {
			App.AppendLog("apply vhosts: " + err.Error())
			return
		}
		App.AppendLog("vhosts applied — hosts file and server configs updated")
		RestartWebServerIfRunning()
	}()
}

// ensureStackEssentials installs anything missing from the essential stack,
// starts it and opens the welcome page.
func EnsureStackEssentials() {
	for _, name := range essentialServicesActive() {
		ms := App.findService(name)
		if ms == nil {
			continue
		}

		if _, ok := packages.DownloadCatalog[name]; ok && !packages.IsInstalled(name, App.BaseDir) {
			if err := packages.DownloadAndInstallVersion(name, ms.Conf.ActiveVersion, App.BaseDir, App.AppendLog, uiDownloadProgress); err != nil {
				App.AppendLog(fmt.Sprintf("[%s] install failed: %v", name, err))
				continue
			}
		}

		if ms.Service == nil || !ms.Conf.Enabled {
			continue
		}
		StartService(ms)
		time.Sleep(400 * time.Millisecond)
	}
	UiRefreshServices()

	time.Sleep(800 * time.Millisecond)
	OpenPath("http://localhost/")
}

func startEssentialStack() {
	for _, name := range essentialServicesActive() {
		if ms := App.findService(name); ms != nil {
			StartService(ms)
		}
	}
}

func StopAllServices() {
	for _, ms := range App.Services {
		if ms.Service != nil {
			_ = ms.Service.Stop()
		}
	}
}

func RestartStack() {
	App.AppendLog("restart stack: stopping all services...")
	StopAllServices()
	process.KillZombieChildren(App.BaseDir)
	time.Sleep(1 * time.Second)
	App.AppendLog("restart stack: starting essentials...")
	EnsureStackEssentials()
	App.AppendLog("restart stack: done")
}

func RunningServiceCount() (running, installed int) {
	for _, ms := range App.Services {
		if ms.Service != nil && ms.Service.Running() {
			running++
		}
		if _, ok := packages.DownloadCatalog[ms.Conf.Name]; ok && packages.IsInstalled(ms.Conf.Name, App.BaseDir) {
			installed++
		}
	}
	return running, installed
}

// startService starts ms, installing it first when its binaries are missing.
// Tools without a process (phpMyAdmin, Composer, ...) open their URL instead.
func StartService(ms *ManagedService) {
	if ms == nil {
		return
	}
	if ms.Service == nil {
		if _, ok := packages.DownloadCatalog[ms.Conf.Name]; ok && !packages.IsInstalled(ms.Conf.Name, App.BaseDir) {
			downloadInBackground(ms.Conf.Name)
			return
		}
		if ms.Conf.OpenURL != "" {
			OpenPath(ms.Conf.OpenURL)
		}
		return
	}

	_, hasCatalog := packages.DownloadCatalog[ms.Conf.Name]
	if hasCatalog && !packages.IsInstalled(ms.Conf.Name, App.BaseDir) {
		name := ms.Conf.Name
		svc := ms.Service
		version := ms.Conf.ActiveVersion
		App.AppendLog(fmt.Sprintf("[%s] not installed — downloading...", name))
		go func() {
			if err := packages.DownloadAndInstallVersion(name, version, App.BaseDir, App.AppendLog, uiDownloadProgress); err != nil {
				App.AppendLog(fmt.Sprintf("[%s] install failed: %v", name, err))
				return
			}
			UiRefreshServices()
			if err := svc.Start(); err != nil {
				App.AppendLog(fmt.Sprintf("[%s] start: %v", name, err))
			}
		}()
		return
	}

	if filepath.IsAbs(ms.Service.ExePath) {
		if _, err := os.Stat(ms.Service.ExePath); err != nil {
			App.AppendLog(fmt.Sprintf("[%s] exe missing: %s", ms.Conf.Name, ms.Service.ExePath))
			return
		}
	}
	if err := ms.Service.Start(); err != nil {
		App.AppendLog(fmt.Sprintf("[%s] start: %v", ms.Conf.Name, err))
	}
}

// toggleService starts a stopped service or stops a running one.
func ToggleService(ms *ManagedService) {
	if IsWebServerName(ms.Conf.Name) && ms.Conf.Name != ActiveWebServer() {
		App.AppendLog(fmt.Sprintf("[%s] not the active web server — switch to %s first", ms.Conf.Name, ms.Conf.Name))
		return
	}
	if s := ms.Service; s != nil && s.Running() {
		go func() { _ = s.Stop() }()
		return
	}
	StartService(ms)
}

func RestartService(ms *ManagedService) {
	s := ms.Service
	if s == nil {
		return
	}
	go func() {
		_ = s.Stop()
		time.Sleep(500 * time.Millisecond)
		if err := s.Start(); err != nil {
			App.AppendLog(fmt.Sprintf("[%s] restart: %v", s.Name, err))
		}
	}()
}

func downloadInBackground(name string) {
	if _, ok := packages.DownloadCatalog[name]; !ok {
		App.AppendLog(fmt.Sprintf("[%s] no download recipe", name))
		return
	}
	go func() {
		if err := packages.DownloadAndInstall(name, App.BaseDir, App.AppendLog, uiDownloadProgress); err != nil {
			App.AppendLog(fmt.Sprintf("[%s] %v", name, err))
			return
		}
		UiRefreshServices()
	}()
}

func UninstallService(ms *ManagedService) {
	if ms.Service != nil && ms.Service.Running() {
		App.AppendLog(fmt.Sprintf("[%s] stop the service before uninstalling", ms.Conf.Name))
		return
	}
	spec, ok := packages.DownloadCatalog[ms.Conf.Name]
	if !ok {
		App.AppendLog(fmt.Sprintf("[%s] no uninstall recipe", ms.Conf.Name))
		return
	}
	installDir := filepath.Join(App.BaseDir, filepath.FromSlash(spec.InstallDir))
	if _, err := os.Stat(installDir); err != nil {
		App.AppendLog(fmt.Sprintf("[%s] not installed at %s", ms.Conf.Name, installDir))
		return
	}
	App.AppendLog(fmt.Sprintf("[%s] removing %s ...", ms.Conf.Name, installDir))
	if err := os.RemoveAll(installDir); err != nil {
		App.AppendLog(fmt.Sprintf("[%s] uninstall: %v", ms.Conf.Name, err))
		return
	}
	App.AppendLog(fmt.Sprintf("[%s] uninstalled", ms.Conf.Name))
	UiRefreshServices()
}

// switchServiceVersion installs the chosen variant and puts things back the
// way they were: a service that was running is started again on the new
// version, and a PHP switch also restarts the web server that loads it.
func SwitchServiceVersion(ms *ManagedService, chosen string) {
	name := ms.Conf.Name
	if chosen == activeVariantLabel(ms) && packages.IsInstalled(name, App.BaseDir) {
		return
	}
	wasRunning := ms.Service != nil && ms.Service.Running()
	go func() {
		if wasRunning {
			App.AppendLog(fmt.Sprintf("[%s] stopping before version switch", name))
			_ = ms.Service.Stop()
		}
		SetStatus(fmt.Sprintf("Switching %s to %s...", name, chosen))
		if err := packages.SetActiveVariant(name, chosen, App.BaseDir, App.AppendLog, uiDownloadProgress); err != nil {
			App.AppendLog(fmt.Sprintf("[%s] switch %s failed: %v", name, chosen, err))
			SetStatus(fmt.Sprintf("%s switch failed", name))
			if wasRunning {
				StartService(ms)
			}
			return
		}
		ms.Conf.ActiveVersion = chosen
		_ = config.SaveConfig(App.BaseDir, App.Cfg)
		App.AppendLog(fmt.Sprintf("[%s] switched to %s", name, chosen))
		SetStatus(fmt.Sprintf("%s is now %s", name, chosen))
		if wasRunning {
			StartService(ms)
		}
		if groupForKind(ms.Conf.Kind) == GroupLanguage && strings.EqualFold(ms.Conf.Kind, "php") {
			RestartWebServerIfRunning()
		}
		UiRefreshServices()
	}()
}

func activeVariantLabel(ms *ManagedService) string {
	spec, ok := packages.DownloadCatalog[ms.Conf.Name]
	if !ok {
		return ""
	}
	current := ms.Conf.ActiveVersion
	for _, v := range spec.Variants {
		if v.Version == current || (current == "" && strings.HasPrefix(spec.Version, v.Version)) {
			return v.Version
		}
	}
	return current
}

var langBinDirs = map[string][]string{
	"Node.js": {"bin/node"},
	"Python":  {"bin/python", "bin/python/Scripts"},
	"Go":      {"bin/go/bin"},
	"Java":    {"bin/java/bin"},
	"Julia":   {"bin/julia/bin"},
	"Zig":     {"bin/zig"},
	"Dart":    {"bin/dart/bin"},
	"Lua":     {"bin/lua"},
	"Ruby":    {"bin/ruby/bin"},
	"Rust":    {"bin/rust/.cargo/bin"},
	"Kotlin":  {"bin/kotlin/bin"},
	"Haskell": {"bin/haskell/bin"},
	"Elixir":  {"bin/elixir/bin", "bin/erlang/bin"},
	"Crystal": {"bin/crystal"},
	"Scala":   {"bin/scala/bin"},
	"Erlang":  {"bin/erlang/bin"},
	"Swift":   {},
}

// openTerminalForService opens a console with the runtime's bin dirs first on
// PATH, preferring Windows Terminal when it is installed.
func OpenTerminalForService(name string) {
	var prepend []string
	for _, rel := range langBinDirs[name] {
		prepend = append(prepend, filepath.Join(App.BaseDir, filepath.FromSlash(rel)))
	}

	env := os.Environ()
	for i, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			env[i] = "PATH=" + strings.Join(append(prepend, e[5:]), ";")
			break
		}
	}
	env = append(env, "GOAMPP_BASE="+App.BaseDir)

	if wt, err := exec.LookPath("wt.exe"); err == nil {
		cmd := exec.Command(wt, "-d", App.BaseDir, "--title", "GoAMPP — "+name, "cmd.exe")
		cmd.Env = env
		if cmd.Start() == nil {
			return
		}
	}
	cmd := exec.Command("cmd.exe")
	cmd.Dir = App.BaseDir
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000010} // CREATE_NEW_CONSOLE
	_ = cmd.Start()
}

func OpenPsqlConsole() {
	pgBin := filepath.Join(App.BaseDir, "bin", "pgsql", "bin")
	if _, err := os.Stat(filepath.Join(pgBin, "psql.exe")); err != nil {
		App.AppendLog("psql: PostgreSQL not installed — install it from Services first")
		return
	}
	cmd := exec.Command("cmd", "/c", "start", "", "cmd", "/K",
		"set PATH="+pgBin+";%PATH% && cd /d "+App.BaseDir+
			" && echo PostgreSQL console — try: psql -U postgres")
	_ = cmd.Start()
}

// ---- configuration ----

func ReloadConfig() error {
	cfg, err := config.LoadConfig(App.BaseDir)
	if err != nil {
		return err
	}
	App.Cfg = cfg
	App.Services = BuildManagedServices(cfg, App.Services)
	return nil
}

func countEnabledServices() int {
	n := 0
	for _, s := range App.Cfg.Services {
		if s.Enabled {
			n++
		}
	}
	return n
}

// ---- startup services ----

func StartupServiceEligible(ms *ManagedService) bool {
	if ms == nil || ms.Service == nil {
		return false
	}
	if (ms.Conf.Name == "Apache" || ms.Conf.Name == "Nginx") && ms.Conf.Name != ActiveWebServer() {
		return false
	}
	if _, ok := packages.DownloadCatalog[ms.Conf.Name]; ok {
		return packages.IsInstalled(ms.Conf.Name, App.BaseDir)
	}
	return true
}

func AutoStartContains(name string) bool {
	if App == nil || App.Cfg == nil {
		return false
	}
	for _, current := range App.Cfg.Settings.AutoStart {
		if current == name {
			return true
		}
	}
	return false
}

func SetServiceAutoStart(name string, enabled bool) error {
	if App == nil || App.Cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}
	before := append([]string(nil), App.Cfg.Settings.AutoStart...)

	next := make([]string, 0, len(before)+1)
	for _, current := range before {
		if current != name {
			next = append(next, current)
		}
	}
	if enabled {
		next = append(next, name)
	}
	App.Cfg.Settings.AutoStart = next
	if err := config.SaveConfig(App.BaseDir, App.Cfg); err != nil {
		App.Cfg.Settings.AutoStart = before
		return err
	}
	return nil
}

// ---- launch with Windows ----

const (
	autoStartKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	autoStartValue   = "GoAMPP"

	flagStartTray = "--tray"
)

func IsAutoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoStartKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(autoStartValue)
	return err == nil
}

func setAutoStart(enable bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, autoStartKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if !enable {
		if err := k.DeleteValue(autoStartValue); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(autoStartValue, fmt.Sprintf(`"%s" %s`, exe, flagStartTray))
}

func ToggleAutoStart() {
	next := !IsAutoStartEnabled()
	if err := setAutoStart(next); err != nil {
		App.AppendLog("auto-start: " + err.Error())
		return
	}
	if next {
		App.AppendLog("auto-start enabled — GoAMPP will launch into the tray on login")
	} else {
		App.AppendLog("auto-start disabled")
	}
}

func HasTrayFlag() bool {
	for _, a := range os.Args[1:] {
		if a == flagStartTray {
			return true
		}
	}
	return false
}

// ---- virtual hosts ----

func ParseVhostInput(name, ext, docroot, portStr, server string) (config.Vhost, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return config.Vhost{}, fmt.Errorf("domain name is required")
	}
	if ext == "" {
		ext = ".test"
	}
	docroot = strings.TrimSpace(docroot)
	if docroot == "" {
		return config.Vhost{}, fmt.Errorf("document root is required")
	}
	portStr = strings.TrimSpace(portStr)
	if portStr == "" {
		portStr = "80"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return config.Vhost{}, fmt.Errorf("invalid port: %s", portStr)
	}
	if server == "" {
		server = "apache"
	}
	return config.Vhost{Domain: name + ext, DocRoot: docroot, Port: port, ServerType: server}, nil
}

func SplitDomain(domain string) (string, string) {
	domain = strings.TrimSpace(domain)
	i := strings.LastIndex(domain, ".")
	if i < 0 {
		return domain, ".test"
	}
	return domain[:i], domain[i:]
}

// ---- projects ----

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func DeleteProject(idx int) {
	if idx < 0 || idx >= len(App.Cfg.Projects) {
		return
	}
	p := App.Cfg.Projects[idx]
	projectDir := filepath.Join(App.BaseDir, "www", p.Name)

	App.AppendLog(fmt.Sprintf("projects: deleting '%s' (%s)...", p.Name, p.Domain))
	if err := os.RemoveAll(projectDir); err != nil {
		App.AppendLog("projects delete: " + err.Error())
	}

	App.Cfg.Projects = append(App.Cfg.Projects[:idx], App.Cfg.Projects[idx+1:]...)
	for i := len(App.Cfg.Vhosts) - 1; i >= 0; i-- {
		if App.Cfg.Vhosts[i].Domain == p.Domain {
			App.Cfg.Vhosts = append(App.Cfg.Vhosts[:i], App.Cfg.Vhosts[i+1:]...)
		}
	}
	_ = config.SaveConfig(App.BaseDir, App.Cfg)
	if err := vhost.ApplyVhosts(App.BaseDir, App.Cfg); err != nil {
		App.AppendLog("projects delete: apply vhosts: " + err.Error())
	}
}

type runtimeStatus struct {
	Label string
	OK    bool
}

func RuntimeStatuses() []runtimeStatus {
	tools := []struct{ label, tool string }{
		{"PHP", "php"}, {"Composer", "composer"}, {"Node.js", "node"},
		{"Python", "python"}, {"Java", "java"}, {"Go", "go"},
	}
	out := make([]runtimeStatus, 0, len(tools))
	for _, t := range tools {
		out = append(out, runtimeStatus{Label: t.label, OK: projects.HasTool(t.tool)})
	}
	return out
}

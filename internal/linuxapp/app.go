//go:build linux

// Package linuxapp provides the initial native Linux desktop experience.
// Packages and daemons are managed by the host distribution; GoAMPP never
// downloads Windows executables or changes system configuration silently.
package linuxapp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type serviceDefinition struct {
	Name     string
	Binaries []string
	Units    []string
}

type serviceState struct {
	Definition serviceDefinition
	Binary     string
	Manager    string
	Unit       string
	Running    bool
}

var serviceDefinitions = []serviceDefinition{
	{Name: "Apache HTTP Server", Binaries: []string{"apache2", "httpd"}, Units: []string{"apache2", "httpd"}},
	{Name: "Nginx", Binaries: []string{"nginx"}, Units: []string{"nginx"}},
	{Name: "PHP-FPM", Binaries: []string{"php-fpm", "php-fpm8.5", "php-fpm8.4", "php-fpm8.3", "php-fpm8.2", "php-fpm8.1", "php-fpm8.0"}, Units: []string{"php-fpm", "php8.5-fpm", "php8.4-fpm", "php8.3-fpm", "php8.2-fpm", "php8.1-fpm", "php8.0-fpm"}},
	{Name: "MariaDB / MySQL", Binaries: []string{"mariadbd", "mysqld"}, Units: []string{"mariadb", "mysql", "mysqld"}},
	{Name: "PostgreSQL", Binaries: []string{"postgres"}, Units: []string{"postgresql"}},
	{Name: "Redis", Binaries: []string{"redis-server"}, Units: []string{"redis-server", "redis"}},
	{Name: "RabbitMQ", Binaries: []string{"rabbitmq-server"}, Units: []string{"rabbitmq-server"}},
}

type toolDefinition struct {
	Name      string
	Binaries  []string
	Arguments []string
}

var toolDefinitions = []toolDefinition{
	{Name: "PHP", Binaries: []string{"php"}, Arguments: []string{"--version"}},
	{Name: "Composer", Binaries: []string{"composer"}, Arguments: []string{"--version"}},
	{Name: "Node.js", Binaries: []string{"node"}, Arguments: []string{"--version"}},
	{Name: "npm", Binaries: []string{"npm"}, Arguments: []string{"--version"}},
	{Name: "Python", Binaries: []string{"python3", "python"}, Arguments: []string{"--version"}},
	{Name: "Go", Binaries: []string{"go"}, Arguments: []string{"version"}},
	{Name: "Java", Binaries: []string{"java"}, Arguments: []string{"-version"}},
	{Name: "Rust", Binaries: []string{"rustc"}, Arguments: []string{"--version"}},
	{Name: "Git", Binaries: []string{"git"}, Arguments: []string{"--version"}},
}

func Run() {
	a := app.NewWithID("cc.thio.goampp.linux")
	w := a.NewWindow("GoAMPP for Linux")
	w.Resize(fyne.NewSize(900, 650))

	projectDir := projectDirectory()
	_ = os.MkdirAll(projectDir, 0o755)

	title := widget.NewLabelWithStyle("GoAMPP for Linux", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel("Native desktop panel for distribution-managed development services.")
	top := container.NewVBox(title, subtitle, widget.NewSeparator())

	serviceList := container.NewVBox()
	serviceStatus := widget.NewLabel("Services are detected from your system PATH and service manager.")
	var refreshServices func()
	var refreshButton *widget.Button
	refreshServices = func() {
		serviceStatus.SetText("Checking installed services...")
		if refreshButton != nil {
			refreshButton.Disable()
		}
		go func() {
			states := discoverServices()
			fyne.Do(func() {
				serviceList.Objects = nil
				for _, state := range states {
					serviceList.Add(makeServiceCard(w, state, refreshServices))
				}
				serviceList.Refresh()
				if len(states) > 0 && states[0].Manager == "" {
					serviceStatus.SetText(fmt.Sprintf("No supported service manager detected. %d service types can still be inspected, but service controls are unavailable.", len(states)))
				} else {
					serviceStatus.SetText(fmt.Sprintf("Showing %d supported service types. GoAMPP does not install or alter system packages without your action.", len(states)))
				}
				if refreshButton != nil {
					refreshButton.Enable()
				}
			})
		}()
	}
	refreshButton = widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), refreshServices)
	servicePage := container.NewBorder(
		container.NewVBox(serviceStatus, refreshButton, widget.NewSeparator()),
		nil, nil, nil, container.NewVScroll(serviceList),
	)

	toolsList := container.NewVBox()
	var refreshTools func()
	refreshTools = func() {
		toolsList.Objects = nil
		for _, tool := range toolDefinitions {
			toolsList.Add(makeToolCard(tool))
		}
		toolsList.Refresh()
	}
	toolsPage := container.NewBorder(
		widget.NewLabel("Developer tools already installed on this Linux system"),
		nil, nil, nil, container.NewVScroll(toolsList),
	)

	rootLabel := widget.NewLabel(projectDir)
	rootLabel.Wrapping = fyne.TextWrapBreak
	projectNameEntry := widget.NewEntry()
	projectNameEntry.SetPlaceHolder("my-first-project")
	templateSelect := widget.NewSelect(projectTemplateNames(), nil)
	templateDescription := widget.NewLabel("")
	updateTemplateDescription := func(name string) {
		if template, ok := projectTemplateByName(name); ok {
			templateDescription.SetText(template.Description)
		}
	}
	templateSelect.OnChanged = updateTemplateDescription
	templateSelect.SetSelected("Static HTML")
	updateTemplateDescription(templateSelect.Selected)
	projectStatus := widget.NewLabel("Create a starter project. Existing folders are never overwritten.")
	projectList := container.NewVBox()
	var refreshProjects func()
	refreshProjects = func() {
		projectList.Objects = nil
		names := listProjects(projectDir)
		if len(names) == 0 {
			projectList.Add(widget.NewLabel("No projects yet. Create one above to get started."))
		} else {
			for _, name := range names {
				projectName := name
				openButton := widget.NewButtonWithIcon("Open folder", theme.FolderOpenIcon(), func() {
					if err := openExternal(filepath.Join(projectDir, projectName)); err != nil {
						dialog.ShowError(err, w)
					}
				})
				actionButton := widget.NewButtonWithIcon("Run", theme.MediaPlayIcon(), func() {
					url, err := startProject(projectDir, projectName)
					if err != nil {
						dialog.ShowError(err, w)
						refreshProjects()
						return
					}
					projectStatus.SetText("Starting " + projectName + " at " + url + ". The preview will open when ready; output is saved in .goampp-run.log.")
					refreshProjects()
					openPreviewWhenReady(projectDir, projectName, url, func(runErr error) {
						fyne.Do(func() {
							if runErr != nil {
								projectStatus.SetText("Could not start " + projectName + ": " + runErr.Error() + ". Check .goampp-run.log.")
							} else {
								projectStatus.SetText("Running " + projectName + " at " + url + ".")
							}
							refreshProjects()
						})
					})
				})
				if _, err := projectRunSpecForDir(filepath.Join(projectDir, projectName)); err != nil {
					actionButton.SetText("No runner")
					actionButton.SetIcon(theme.InfoIcon())
					actionButton.Disable()
				} else if isProjectRunning(projectDir, projectName) {
					actionButton.SetText("Stop")
					actionButton.SetIcon(theme.MediaStopIcon())
					actionButton.OnTapped = func() {
						actionButton.Disable()
						actionButton.SetText("Stopping…")
						go func() {
							err := stopProject(projectDir, projectName)
							fyne.Do(func() {
								if err != nil {
								projectStatus.SetText("Could not stop " + projectName + ": " + err.Error())
									dialog.ShowError(err, w)
								} else {
									projectStatus.SetText("Stopped " + projectName + ".")
								}
								refreshProjects()
							})
						}()
					}
				}
				actions := []fyne.CanvasObject{actionButton}
				logPath := filepath.Join(projectDir, projectName, ".goampp-run.log")
				if _, err := os.Stat(logPath); err == nil {
					logButton := widget.NewButton("Logs", func() {
						if err := openExternal(logPath); err != nil {
							dialog.ShowError(err, w)
						}
					})
					actions = append(actions, logButton)
				}
				actions = append(actions, openButton)
				projectList.Add(container.NewPadded(container.NewBorder(
					nil, widget.NewSeparator(), nil, container.NewHBox(actions...),
					widget.NewLabelWithStyle(projectName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				)))
			}
		}
		projectList.Refresh()
	}
	createButton := widget.NewButtonWithIcon("Create project", theme.ContentAddIcon(), func() {
		created, err := createProject(projectDir, projectNameEntry.Text, templateSelect.Selected)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		projectStatus.SetText("Created " + created + "  •  See README.md for run instructions.")
		projectNameEntry.SetText("")
		refreshProjects()
		dialog.ShowInformation("Project created", "Your starter files are ready. Open the project folder or read README.md for the next steps.", w)
	})
	projectHeader := container.NewVBox(
		widget.NewLabelWithStyle("Projects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		rootLabel,
		container.NewHBox(
			widget.NewButtonWithIcon("Open projects folder", theme.FolderOpenIcon(), func() {
				if err := openExternal(projectDir); err != nil {
					dialog.ShowError(err, w)
				}
			}),
			widget.NewButtonWithIcon("Open localhost", theme.ComputerIcon(), func() {
				if err := openExternal("http://localhost/"); err != nil {
					dialog.ShowError(err, w)
				}
			}),
			widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), refreshProjects),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Create a new project", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewVBox(
			container.NewGridWithColumns(2,
				widget.NewLabel("Project name"), projectNameEntry,
				widget.NewLabel("Starter template"), templateSelect,
			),
			templateDescription,
		),
		createButton,
		projectStatus,
		widget.NewSeparator(),
		widget.NewLabel("Install web servers, databases, and runtimes with your distribution's package manager. GoAMPP does not install system packages automatically."),
		widget.NewButton("Show installation command", func() {
			command := suggestedInstallCommand()
			entry := widget.NewEntry()
			entry.SetText(command)
			entry.MultiLine = true
			entry.Wrapping = fyne.TextWrapBreak
			copyButton := widget.NewButton("Copy command", func() {
				w.Clipboard().SetContent(entry.Text)
				dialog.ShowInformation("Copied", "Installation command copied to clipboard.", w)
			})
			dialog.ShowCustom("Native package command", "Close", container.NewVBox(entry, copyButton), w)
		}),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Your projects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	projectsPage := container.NewBorder(projectHeader, nil, nil, nil, container.NewVScroll(projectList))
	refreshProjects()

	tabs := container.NewAppTabs(
		container.NewTabItem("Services", servicePage),
		container.NewTabItem("Developer tools", toolsPage),
		container.NewTabItem("Projects", container.NewPadded(projectsPage)),
	)
	w.SetContent(container.NewBorder(top, nil, nil, nil, tabs))

	refreshServices()
	refreshTools()
	w.SetCloseIntercept(func() {
		stopAllProjects()
		a.Quit()
	})
	w.ShowAndRun()
}

func makeServiceCard(w fyne.Window, state serviceState, refresh func()) fyne.CanvasObject {
	detail := "Not detected"
	if state.Unit != "" {
		detail = "service unit: " + state.Unit
		if state.Binary != "" {
			detail += "  •  " + state.Binary
		}
	} else if state.Binary != "" {
		detail = state.Binary + "  •  no matching service unit found"
	}

	status := "Not installed"
	switch {
	case state.Unit != "" && state.Running:
		status = "Running"
	case state.Unit != "":
		status = "Stopped"
	case state.Binary != "":
		status = "Installed"
	}

	info := container.NewVBox(
		widget.NewLabelWithStyle(state.Definition.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(detail),
		widget.NewLabel("Status: "+status),
	)
	var action *widget.Button
	action = widget.NewButton("Start", func() {
		runServiceAction(w, action, state.Manager, state.Unit, "start", refresh)
	})
	action.SetIcon(theme.MediaPlayIcon())
	if state.Running {
		action.SetText("Stop")
		action.SetIcon(theme.MediaStopIcon())
		action.OnTapped = func() {
			runServiceAction(w, action, state.Manager, state.Unit, "stop", refresh)
		}
	}
	if state.Unit == "" {
		action.SetText("No service unit")
		action.SetIcon(theme.InfoIcon())
		action.Disable()
	}
	return container.NewPadded(container.NewBorder(nil, widget.NewSeparator(), nil, action, info))
}

func runServiceAction(w fyne.Window, action *widget.Button, manager, unit, verb string, refresh func()) {
	action.Disable()
	switch verb {
	case "start":
		action.SetText("Starting…")
	case "stop":
		action.SetText("Stopping…")
	}
	go func() {
		err := controlUnit(manager, unit, verb)
		fyne.Do(func() {
			if err != nil {
				dialog.ShowError(err, w)
			}
			refresh()
		})
	}()
}

func makeToolCard(tool toolDefinition) fyne.CanvasObject {
	binary := findExecutable(tool.Binaries)
	status := "Not installed"
	if binary != "" {
		status = binary
	}
	statusLabel := widget.NewLabel(status)
	statusLabel.Wrapping = fyne.TextWrapBreak
	if binary != "" && len(tool.Arguments) > 0 {
		go func(binary string, args []string, label *widget.Label) {
			output, err := runCommandWithTimeout(2*time.Second, binary, args...)
			if err != nil {
				return
			}
			for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				version := line
				fyne.Do(func() {
					label.SetText(binary + "\n" + version)
				})
				return
			}
		}(binary, append([]string(nil), tool.Arguments...), statusLabel)
	}
	return container.NewPadded(container.NewBorder(
		nil, widget.NewSeparator(), nil, statusLabel,
		widget.NewLabelWithStyle(tool.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	))
}

func runCommandWithTimeout(timeout time.Duration, command string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, command, args...).CombinedOutput()
}

func discoverServices() []serviceState {
	manager := detectServiceManager()
	units := listSystemdUnits()
	initScripts := listInitScripts()
	states := make([]serviceState, 0, len(serviceDefinitions))
	for _, def := range serviceDefinitions {
		state := serviceState{Definition: def, Binary: findExecutable(def.Binaries), Manager: manager}
		for _, candidate := range def.Units {
			name := strings.TrimSuffix(candidate, ".service")
			if manager == "systemd" {
				if _, ok := units[name+".service"]; ok {
					state.Unit = name
					break
				}
			} else if manager == "openrc" || manager == "sysvinit" {
				if _, ok := initScripts[name]; ok {
					state.Unit = name
					break
				}
			}
		}
		if def.Name == "PHP-FPM" && state.Unit == "" {
			state.Unit = findPHPFPMUnit(manager, units, initScripts)
		}
		if state.Unit != "" {
			state.Running = serviceIsRunning(manager, state.Unit)
		}
		states = append(states, state)
	}
	return states
}

func findPHPFPMUnit(manager string, systemdUnits, initScripts map[string]struct{}) string {
	var candidates []string
	switch manager {
	case "systemd":
		for unit := range systemdUnits {
			if strings.HasPrefix(unit, "php") && strings.HasSuffix(unit, "-fpm.service") {
				candidates = append(candidates, strings.TrimSuffix(unit, ".service"))
			}
		}
	case "openrc", "sysvinit":
		for unit := range initScripts {
			if strings.HasPrefix(unit, "php") && strings.HasSuffix(unit, "-fpm") {
				candidates = append(candidates, unit)
			}
		}
	default:
		return ""
	}
	sort.Strings(candidates)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[len(candidates)-1]
}

func detectServiceManager() string {
	_, systemctlErr := exec.LookPath("systemctl")
	_, rcServiceErr := exec.LookPath("rc-service")
	_, serviceErr := exec.LookPath("service")
	_, systemdErr := os.Stat("/run/systemd/system")
	_, initScriptsErr := os.Stat("/etc/init.d")
	return chooseServiceManager(systemctlErr == nil && systemdErr == nil, rcServiceErr == nil && initScriptsErr == nil, serviceErr == nil && initScriptsErr == nil)
}

func chooseServiceManager(systemdActive, openRCAvailable, sysVAvailable bool) string {
	switch {
	case systemdActive:
		return "systemd"
	case openRCAvailable:
		return "openrc"
	case sysVAvailable:
		return "sysvinit"
	default:
		return ""
	}
}

func listSystemdUnits() map[string]struct{} {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return map[string]struct{}{}
	}
	out, err := runCommandWithTimeout(5*time.Second, "systemctl", "list-unit-files", "--type=service", "--no-legend", "--no-pager")
	if err != nil {
		return map[string]struct{}{}
	}
	return parseSystemdUnitFiles(string(out))
}

func parseSystemdUnitFiles(output string) map[string]struct{} {
	units := make(map[string]struct{})
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.HasSuffix(fields[0], ".service") {
			units[fields[0]] = struct{}{}
		}
	}
	return units
}

func listInitScripts() map[string]struct{} {
	units := make(map[string]struct{})
	entries, err := os.ReadDir("/etc/init.d")
	if err != nil {
		return units
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			units[entry.Name()] = struct{}{}
		}
	}
	return units
}

func serviceIsRunning(manager, unit string) bool {
	command, args, err := serviceCommand(manager, unit, "status")
	if err != nil {
		return false
	}
	_, err = runCommandWithTimeout(2*time.Second, command, args...)
	return err == nil
}

func serviceCommand(manager, unit, action string) (string, []string, error) {
	if unit == "" {
		return "", nil, fmt.Errorf("service name is required")
	}
	switch action {
	case "start", "stop", "status":
	default:
		return "", nil, fmt.Errorf("unsupported service action %q", action)
	}
	switch manager {
	case "systemd":
		return "systemctl", []string{action, unit}, nil
	case "openrc":
		return "rc-service", []string{unit, action}, nil
	case "sysvinit":
		return "service", []string{unit, action}, nil
	default:
		return "", nil, fmt.Errorf("no supported service manager was detected")
	}
}

func controlUnit(manager, unit, action string) error {
	command, args, err := serviceCommand(manager, unit, action)
	if err != nil {
		return err
	}
	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("%s is not available; install or use your distribution's service manager", command)
	}
	output, err := runCommandWithTimeout(30*time.Second, command, args...)
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("%s %s %s failed: %s. Your account may need authorization to control this system service", command, action, unit, message)
	}
	return nil
}

func findExecutable(candidates []string) string {
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}

func projectDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", "GoAMPP", "www")
	}
	return filepath.Join(home, "GoAMPP", "www")
}

func desktopOpenCommand(target string, lookup func(string) (string, error)) (string, []string, error) {
	candidates := []struct {
		name string
		args []string
	}{
		{name: "xdg-open"},
		{name: "gio", args: []string{"open"}},
		{name: "kde-open5"},
		{name: "kde-open"},
	}
	for _, candidate := range candidates {
		opener, err := lookup(candidate.name)
		if err != nil {
			continue
		}
		args := append(append([]string(nil), candidate.args...), target)
		return opener, args, nil
	}
	return "", nil, fmt.Errorf("no supported desktop opener was found; install xdg-utils or open manually: %s", target)
}

func openExternal(target string) error {
	opener, args, err := desktopOpenCommand(target, exec.LookPath)
	if err != nil {
		return err
	}
	cmd := exec.Command(opener, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open %s: %w", target, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func suggestedInstallCommand() string {
	if _, err := exec.LookPath("pacman"); err == nil {
		return "sudo pacman -S --needed apache nginx php-fpm mariadb postgresql redis nodejs npm python python-pip go jdk-openjdk composer"
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		return "sudo dnf install httpd nginx php-fpm mariadb-server postgresql-server redis nodejs npm python3 golang java-latest-openjdk composer"
	}
	if _, err := exec.LookPath("zypper"); err == nil {
		return "sudo zypper install apache2 nginx php-fpm mariadb postgresql-server redis nodejs npm python3 go java-17-openjdk-devel composer"
	}
	if _, err := exec.LookPath("apt-get"); err == nil {
		return "sudo apt install apache2 nginx php-fpm mariadb-server postgresql redis-server php-cli composer nodejs npm python3 python3-pip golang-go default-jdk"
	}
	return "# Install Apache/Nginx, PHP-FPM, MariaDB/PostgreSQL, Redis and your preferred runtimes using your Linux distribution's package manager."
}

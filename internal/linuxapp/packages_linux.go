//go:build linux

package linuxapp

import (
	"fmt"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type linuxPackageNames struct {
	APT    string
	Pacman string
	DNF    string
	Zypper string
}

var linuxServicePackages = map[string]linuxPackageNames{
	"Apache HTTP Server": {APT: "apache2", Pacman: "apache", DNF: "httpd", Zypper: "apache2"},
	"Nginx":              {APT: "nginx", Pacman: "nginx", DNF: "nginx", Zypper: "nginx"},
	"PHP-FPM":            {APT: "php-fpm", Pacman: "php", DNF: "php-fpm", Zypper: "php8-fpm"},
	"MariaDB / MySQL":    {APT: "mariadb-server", Pacman: "mariadb", DNF: "mariadb-server", Zypper: "mariadb"},
	"PostgreSQL":         {APT: "postgresql", Pacman: "postgresql", DNF: "postgresql-server", Zypper: "postgresql-server"},
	"Redis":              {APT: "redis-server", Pacman: "redis", DNF: "redis", Zypper: "redis"},
	"RabbitMQ":           {APT: "rabbitmq-server", Pacman: "rabbitmq", DNF: "rabbitmq-server", Zypper: "rabbitmq-server"},
}

var linuxToolPackages = map[string]linuxPackageNames{
	"PHP":      {APT: "php-cli", Pacman: "php", DNF: "php-cli", Zypper: "php8"},
	"Composer": {APT: "composer", Pacman: "composer", DNF: "composer", Zypper: "composer"},
	"Node.js":  {APT: "nodejs", Pacman: "nodejs", DNF: "nodejs", Zypper: "nodejs"},
	"npm":      {APT: "npm", Pacman: "npm", DNF: "npm", Zypper: "npm"},
	"Python":   {APT: "python3", Pacman: "python", DNF: "python3", Zypper: "python3"},
	"Go":       {APT: "golang-go", Pacman: "go", DNF: "golang", Zypper: "go"},
	"Java":     {APT: "default-jdk", Pacman: "jdk-openjdk", DNF: "java-latest-openjdk-devel", Zypper: "java-17-openjdk-devel"},
	"Rust":     {APT: "rustc cargo", Pacman: "rust", DNF: "rust cargo", Zypper: "rust cargo"},
	"Git":      {APT: "git", Pacman: "git", DNF: "git", Zypper: "git"},
}

func linuxPackageCommand(manager string, packages linuxPackageNames) (string, error) {
	var names string
	var command string
	switch manager {
	case "apt":
		command, names = "sudo apt install", packages.APT
	case "pacman":
		command, names = "sudo pacman -S --needed", packages.Pacman
	case "dnf":
		command, names = "sudo dnf install", packages.DNF
	case "zypper":
		command, names = "sudo zypper install", packages.Zypper
	default:
		return "", fmt.Errorf("unsupported package manager %q", manager)
	}
	if strings.TrimSpace(names) == "" {
		return "", fmt.Errorf("no package mapping is defined for %s", manager)
	}
	return command + " " + names, nil
}

func detectLinuxPackageManager() string {
	for _, candidate := range []struct {
		name string
		bin  string
	}{
		{"pacman", "pacman"},
		{"dnf", "dnf"},
		{"zypper", "zypper"},
		{"apt", "apt-get"},
	} {
		if _, err := exec.LookPath(candidate.bin); err == nil {
			return candidate.name
		}
	}
	return ""
}

func suggestedLinuxInstallCommand(name string, isTool bool) (string, error) {
	manager := detectLinuxPackageManager()
	if manager == "" {
		return "", fmt.Errorf("no supported package manager was detected; use your distribution's software installer")
	}
	packages := linuxServicePackages
	if isTool {
		packages = linuxToolPackages
	}
	definition, ok := packages[name]
	if !ok {
		return "", fmt.Errorf("there is no package mapping for %s", name)
	}
	return linuxPackageCommand(manager, definition)
}

func showLinuxPackageCommand(w fyne.Window, name string, isTool bool) {
	command, err := suggestedLinuxInstallCommand(name, isTool)
	if err != nil {
		dialog.ShowInformation("Install "+name, err.Error(), w)
		return
	}
	commandLabel := widget.NewLabel(command)
	commandLabel.Wrapping = fyne.TextWrapBreak
	commandLabel.TextStyle = fyne.TextStyle{Monospace: true}
	commandLabel.Selectable = true
	copyButton := widget.NewButtonWithIcon("Copy command", theme.ContentCopyIcon(), func() {
		w.Clipboard().SetContent(command)
		linuxSetStatus("Install command copied for " + name)
	})
	note := widget.NewLabel("This is a suggested command only. Review it, then run it in a terminal if you trust your distribution's package source. GoAMPP will not run sudo or install packages on its own.")
	note.Wrapping = fyne.TextWrapWord
	content := container.NewVBox(note, linuxVSpace(8), container.NewBorder(nil, nil, nil, copyButton, commandLabel))
	d := dialog.NewCustom("Install "+name, "Close", content, w)
	d.Resize(fyne.NewSize(560, 190))
	d.Show()
}

//go:build linux

package linuxapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func linuxServiceConfigPaths(name string) []string {
	patterns := map[string][]string{
		"Apache HTTP Server": {
			"/etc/apache2/apache2.conf", "/etc/apache2/ports.conf",
			"/etc/apache2/sites-available/*.conf", "/etc/httpd/conf/httpd.conf",
			"/etc/httpd/conf.d/*.conf",
		},
		"Nginx": {
			"/etc/nginx/nginx.conf", "/etc/nginx/sites-available/*",
			"/etc/nginx/conf.d/*.conf", "/etc/nginx/conf.d/*",
		},
		"Caddy": {"/etc/caddy/Caddyfile"},
		"PHP-FPM": {
			"/etc/php/*/fpm/php.ini", "/etc/php/*/fpm/pool.d/*.conf",
			"/etc/php-fpm.conf", "/etc/php-fpm.d/*.conf",
		},
		"MariaDB / MySQL": {
			"/etc/my.cnf", "/etc/mysql/my.cnf", "/etc/mysql/mariadb.conf.d/*.cnf",
			"/etc/mysql/mysql.conf.d/*.cnf",
		},
		"PostgreSQL": {
			"/etc/postgresql/*/main/postgresql.conf", "/etc/postgresql/*/main/pg_hba.conf",
			"/var/lib/pgsql/data/postgresql.conf",
		},
		"Redis": {"/etc/redis/redis.conf", "/etc/redis.conf"},
		"Memcached": {"/etc/memcached.conf", "/etc/sysconfig/memcached"},
		"RabbitMQ": {"/etc/rabbitmq/rabbitmq.conf", "/etc/rabbitmq/advanced.config"},
		"Docker Engine": {"/etc/docker/daemon.json"},
	}
	seen := make(map[string]struct{})
	var result []string
	for _, pattern := range patterns[name] {
		matches, _ := filepath.Glob(pattern)
		for _, path := range matches {
			info, err := os.Stat(path)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			path = filepath.Clean(path)
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			result = append(result, path)
		}
	}
	sort.Strings(result)
	return result
}

func linuxServiceActionCommand(manager, unit, action string) (string, error) {
	command, args, err := serviceCommand(manager, unit, action)
	if err != nil {
		return "", err
	}
	if action == "start" || action == "stop" || action == "restart" {
		return "sudo " + command + " " + strings.Join(args, " "), nil
	}
	return command + " " + strings.Join(args, " "), nil
}

func showLinuxServiceLogs(w fyne.Window, state serviceState) {
	logText := widget.NewLabel("Loading recent service logs…")
	logText.Wrapping = fyne.TextWrapBreak
	logText.TextStyle = fyne.TextStyle{Monospace: true}
	refresh := func() {
		logText.SetText("Loading recent service logs…")
		go func() {
			output, err := runCommandWithTimeout(5*time.Second, "journalctl", "--no-pager", "-n", "200", "-u", state.Unit)
			result := strings.TrimSpace(string(output))
			if err != nil {
				result += "\n\nCould not read the system journal: " + err.Error()
				result += "\nYou may need to inspect it manually with: sudo journalctl -u " + state.Unit + " -n 200 --no-pager"
			}
			if result == "" {
				result = "The journal returned no messages for this unit."
			}
			if len(result) > 256*1024 {
				result = "[Only the last 256 KiB are shown.]\n\n" + result[len(result)-256*1024:]
			}
			fyne.Do(func() { logText.SetText(result) })
		}()
	}
	refreshButton := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), refresh)
	copyButton := widget.NewButtonWithIcon("Copy journal command", theme.ContentCopyIcon(), func() {
		w.Clipboard().SetContent("sudo journalctl -u " + state.Unit + " -n 200 --no-pager")
		linuxSetStatus("Journal command copied for " + state.Unit)
	})
	content := container.NewBorder(
		container.NewHBox(refreshButton, copyButton),
		nil, nil, nil,
		container.NewScroll(logText),
	)
	d := dialog.NewCustom("Service logs · "+state.Unit, "Close", content, w)
	d.Resize(fyne.NewSize(820, 520))
	d.Show()
	refresh()
}

func showLinuxServiceActions(w fyne.Window, state serviceState, refresh func()) {
	title := widget.NewLabelWithStyle(state.Definition.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	status := "Not installed"
	if state.Binary != "" {
		status = "Installed at " + state.Binary
	}
	if state.Unit != "" {
		if state.Running {
			status = "Running · " + state.Unit
		} else {
			status = "Stopped · " + state.Unit
		}
	}
	statusLabel := widget.NewLabel(status)
	statusLabel.Wrapping = fyne.TextWrapBreak
	unitLabel := widget.NewLabel("Service manager: " + fallbackLinuxValue(state.Manager, "not detected") + "\nUnit: " + fallbackLinuxValue(state.Unit, "not detected"))
	unitLabel.Wrapping = fyne.TextWrapBreak
	openBinary := widget.NewButtonWithIcon("Open binary folder", theme.FolderOpenIcon(), func() {
		if state.Binary == "" {
			dialog.ShowInformation("Service", "No executable was detected for this service.", w)
			return
		}
		if err := openExternal(filepath.Dir(state.Binary)); err != nil {
			dialog.ShowError(err, w)
		}
	})
	copyStatus := widget.NewButtonWithIcon("Copy status command", theme.ContentCopyIcon(), func() {
		command, err := linuxServiceActionCommand(state.Manager, state.Unit, "status")
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		w.Clipboard().SetContent(command)
		linuxSetStatus("Status command copied for " + state.Definition.Name)
	})
	buttons := []fyne.CanvasObject{openBinary, copyStatus}
	if state.Unit != "" {
		restart := widget.NewButtonWithIcon("Restart", theme.MediaReplayIcon(), nil)
		restart.OnTapped = func() {
			runServiceAction(w, restart, state.Manager, state.Unit, "restart", refresh)
		}
		logs := widget.NewButtonWithIcon("Recent logs", theme.DocumentIcon(), func() {
			if state.Manager != "systemd" {
				dialog.ShowInformation("Service logs", "Automatic journal viewing is available for systemd services. Copy the command and inspect the distribution logs for other service managers.", w)
			}
			showLinuxServiceLogs(w, state)
		})
		copyRestart := widget.NewButtonWithIcon("Copy restart command", theme.ContentCopyIcon(), func() {
			command, err := linuxServiceActionCommand(state.Manager, state.Unit, "restart")
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			w.Clipboard().SetContent(command)
			linuxSetStatus("Restart command copied for " + state.Definition.Name)
		})
		buttons = append(buttons, restart, logs, copyRestart)
	} else if state.Binary == "" {
		buttons = append(buttons, widget.NewButtonWithIcon("Install guide", theme.DownloadIcon(), func() {
			showLinuxPackageCommand(w, state.Definition.Name, false)
		}))
	}

	configPaths := linuxServiceConfigPaths(state.Definition.Name)
	configSelect := widget.NewSelect(configPaths, nil)
	if len(configPaths) > 0 {
		configSelect.SetSelected(configPaths[0])
	}
	openConfig := widget.NewButtonWithIcon("Open config", theme.FolderOpenIcon(), func() {
		if configSelect.Selected == "" {
			dialog.ShowInformation("Configuration", "No known configuration file was found for this component. You can open a file manually from Config Editor.", w)
			return
		}
		if err := openExternal(configSelect.Selected); err != nil {
			dialog.ShowError(err, w)
		}
	})
	if len(configPaths) == 0 {
		configSelect.Disable()
	}
	configRow := container.NewBorder(nil, nil, widget.NewLabel("Config file"), openConfig, configSelect)
	commands := container.NewGridWithColumns(2, buttons...)
	content := container.NewVBox(
		title,
		statusLabel,
		unitLabel,
		linuxVSpace(6),
		commands,
		linuxVSpace(8),
		linuxBoldText("Configuration", 12.5),
		configRow,
		widget.NewLabel("Open actions use your desktop's configured editor or file manager. System files may be read-only without administrator permissions."),
	)
	d := dialog.NewCustom("Service details", "Close", content, w)
	d.Resize(fyne.NewSize(700, 380))
	d.Show()
}

func fallbackLinuxValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func validateLinuxServiceActionCommand(manager, unit, action string) error {
	if manager == "" || unit == "" {
		return fmt.Errorf("a detected service manager and unit are required")
	}
	_, err := linuxServiceActionCommand(manager, unit, action)
	return err
}

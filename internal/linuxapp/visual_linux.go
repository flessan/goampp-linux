//go:build linux

package linuxapp

import (
	"image/color"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"goampp/internal/uikit"
)

var (
	linuxToolProbeMu        sync.Mutex
	linuxToolVersionCache   = map[string]string{}
	linuxToolProbePending   = map[string]bool{}
	linuxToolProbeWaiters   = map[string][]*widget.Label{}
	linuxToolProbeSemaphore = make(chan struct{}, 4)
)

func linuxCachedToolVersion(name string) (string, bool) {
	linuxToolProbeMu.Lock()
	defer linuxToolProbeMu.Unlock()
	version, ok := linuxToolVersionCache[name]
	return version, ok
}

func linuxRequestToolVersion(name, binary string, args []string, label *widget.Label) {
	linuxToolProbeMu.Lock()
	if version, ok := linuxToolVersionCache[name]; ok {
		linuxToolProbeMu.Unlock()
		label.SetText(binary + " · " + version)
		return
	}
	linuxToolProbeWaiters[name] = append(linuxToolProbeWaiters[name], label)
	if linuxToolProbePending[name] {
		linuxToolProbeMu.Unlock()
		return
	}
	linuxToolProbePending[name] = true
	linuxToolProbeMu.Unlock()

	go func() {
		linuxToolProbeSemaphore <- struct{}{}
		defer func() { <-linuxToolProbeSemaphore }()
		output, err := runCommandWithTimeout(2*time.Second, binary, args...)
		version := ""
		if err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					version = line
					break
				}
			}
		}
		fyne.Do(func() {
			linuxToolProbeMu.Lock()
			delete(linuxToolProbePending, name)
			waiters := append([]*widget.Label(nil), linuxToolProbeWaiters[name]...)
			delete(linuxToolProbeWaiters, name)
			if version != "" {
				linuxToolVersionCache[name] = version
			}
			linuxToolProbeMu.Unlock()
			if version != "" {
				for _, waiter := range waiters {
					waiter.SetText(binary + " · " + version)
				}
			}
		})
	}()
}

func linuxResetToolVersionCache() {
	linuxToolProbeMu.Lock()
	linuxToolVersionCache = make(map[string]string)
	linuxToolProbeMu.Unlock()
}

var linuxServiceAssets = map[string]string{
	"Apache HTTP Server": "icons/apache.ico",
	"Nginx":              "logos/NGINX.png",
	"PHP-FPM":            "logos/PHP.png",
	"MariaDB / MySQL":    "logos/MySQL.png",
	"PostgreSQL":         "logos/PostgresSQL.png",
	"Redis":              "logos/Redis.png",
	"RabbitMQ":           "icons/rabbitmq.ico",
}

var linuxToolAssets = map[string]string{
	"PHP":               "logos/PHP.png",
	"Composer":          "logos/Composer.png",
	"Node.js":           "logos/Node.js.png",
	"npm":               "logos/Node.js.png",
	"Python":            "logos/Python.png",
	"Go":                "logos/Go.png",
	"Java":              "logos/Java.png",
	"Rust":              "logos/Rust.png",
	"Cargo":             "logos/Rust.png",
	"Ruby":              "logos/Ruby.png",
	"Bundler":           "logos/Ruby.png",
	"Elixir":            "logos/Elixir.png",
	"Erlang":            "logos/erlang.png",
	"Maven":             "logos/Java.png",
	"Gradle":            "logos/Java.png",
	"PostgreSQL client": "logos/PostgresSQL.png",
}

func linuxAsset(name string) fyne.Resource { return uikit.Asset(name) }

func linuxLogoResource() fyne.Resource               { return linuxAsset("logo.png") }
func linuxServiceResource(name string) fyne.Resource { return linuxAsset(linuxServiceAssets[name]) }
func linuxToolResource(name string) fyne.Resource    { return linuxAsset(linuxToolAssets[name]) }

func linuxImage(res fyne.Resource, size float32, fallbackName string) fyne.CanvasObject {
	return uikit.Avatar(res, fallbackName, size)
}

func linuxLetterMark(name string, size float32) fyne.CanvasObject {
	return uikit.Avatar(nil, name, size)
}

func linuxPad(obj fyne.CanvasObject, top, bottom, left, right float32) fyne.CanvasObject {
	return uikit.Pad(obj, top, bottom, left, right)
}

func linuxSurface(content fyne.CanvasObject, top, bottom, left, right float32) fyne.CanvasObject {
	return uikit.Surface(content, top, bottom, left, right)
}

func linuxGap(height float32) fyne.CanvasObject {
	return uikit.Gap(1, height)
}

func linuxHGap(width float32) fyne.CanvasObject {
	return uikit.Gap(width, 1)
}

func linuxStatusBadge(label string, value color.Color) fyne.CanvasObject {
	dot := canvas.NewCircle(value)
	dotBox := container.NewGridWrap(fyne.NewSize(6, 6), dot)
	text := canvas.NewText(label, theme.Color(linuxColorMuted))
	text.TextSize = 11.5
	return container.NewHBox(container.NewCenter(dotBox), text)
}

func linuxPageIntro(title, subtitle string, icon fyne.Resource) fyne.CanvasObject {
	iconView := container.NewGridWrap(fyne.NewSize(42, 42), widget.NewIcon(icon))
	heading := container.NewVBox(
		widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(subtitle),
	)
	return linuxSurface(container.NewHBox(iconView, linuxHGap(12), heading), 10, 10, 12, 12)
}

type linuxSegmented struct {
	shared *uikit.Segmented
	obj    fyne.CanvasObject
}

func newLinuxSegmented(options []string, onChange func(int)) *linuxSegmented {
	control := uikit.NewSegmented(options, onChange)
	return &linuxSegmented{shared: control, obj: control.Object()}
}

func (s *linuxSegmented) setSelected(index int, notify bool) {
	s.shared.SetSelected(index, notify)
}

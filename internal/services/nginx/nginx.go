//go:build windows

// Package nginx registers Nginx with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package nginx

import (
	"fmt"
	"regexp"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Nginx", packages.DownloadSpec{
		Version:    "1.28.3 stable",
		URL:        "https://nginx.org/download/nginx-1.28.3.zip",
		FileName:   "nginx-1.28.3.zip",
		InstallDir: "bin/nginx",
		StripTop:   "nginx-1.28.3/",
		Kind:       packages.Zip,
		CheckFile:  "nginx.exe",
	}, resolveNginxStable)
}

func resolveNginxStable(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	const pageURL = "https://nginx.org/en/download.html"
	log("  resolving latest nginx stable Windows build ...")
	body, err := packages.FetchReleaseText(pageURL)
	if err != nil {
		return "", "", "", "", err
	}

	re := regexp.MustCompile(`(?is)Stable version.*?nginx/Windows-([0-9]+\.[0-9]+\.[0-9]+)`)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return "", "", "", "", fmt.Errorf("nginx stable Windows version not found")
	}
	v := m[1]
	file := "nginx-" + v + ".zip"
	return "https://nginx.org/download/" + file, file, "nginx-" + v + "/", v + " stable", nil
}

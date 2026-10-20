//go:build windows

// Package julia registers Julia with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package julia

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Julia", packages.DownloadSpec{
		Version:    "1.11.5",
		URL:        "https://julialang-s3.julialang.org/bin/winnt/x64/1.11/julia-1.11.5-win64.zip",
		FileName:   "julia-1.11.5-win64.zip",
		InstallDir: "bin/julia",
		StripTop:   "julia-1.11.5/",
		Kind:       packages.Zip,
		CheckFile:  "bin/julia.exe",
		Notes:      "Julia — high-performance scientific computing language. REPL + package manager included.",
	}, resolveJuliaLatest)
}

func resolveJuliaLatest(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	const apiURL = "https://julialang-s3.julialang.org/bin/versions.json"
	log("  resolving latest stable Julia Windows x64 archive ...")
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", "", "", err
	}
	req.Header.Set("User-Agent", "GoAMPP")
	resp, err := packages.LatestHTTPClient.Do(req)
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", "", "", fmt.Errorf("Julia versions API returned HTTP %d", resp.StatusCode)
	}
	var versions map[string]struct {
		Stable bool `json:"stable"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return "", "", "", "", err
	}
	latest := ""
	stableRe := regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	for v, info := range versions {
		if !info.Stable || !stableRe.MatchString(v) {
			continue
		}
		if latest == "" || packages.NumericVersionGT(v, latest) {
			latest = v
		}
	}
	if latest == "" {
		return "", "", "", "", fmt.Errorf("no stable Julia version found")
	}
	parts := strings.Split(latest, ".")
	series := parts[0] + "." + parts[1]
	file := "julia-" + latest + "-win64.zip"
	dlURL := fmt.Sprintf("https://julialang-s3.julialang.org/bin/winnt/x64/%s/%s", series, file)
	return dlURL, file, "julia-" + latest + "/", latest, nil
}

//go:build windows

// Package zig registers Zig with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package zig

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Zig", packages.DownloadSpec{
		Version:     "latest stable",
		URL:         "https://ziglang.org/download/0.14.0/zig-windows-x86_64-0.14.0.zip",
		FileName:    "zig-windows-x86_64-0.14.0.zip",
		InstallDir:  "bin/zig",
		StripTop:    "zig-windows-x86_64-0.14.0/",
		Kind:        packages.Zip,
		CheckFile:   "zig.exe",
		Notes:       "Zig — systems programming language + build system + C compiler.",
		URLResolver: resolveZigLatest,
	}, resolveZigLatest)
}

func resolveZigLatest(log func(string)) (url, fileName, stripTop, version string, err error) {
	log("  resolving latest Zig stable from ziglang.org/download/index.json ...")
	resp, err := http.Get("https://ziglang.org/download/index.json")
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()
	var index map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return "", "", "", "", err
	}
	var versions []string
	for k := range index {
		if k == "master" {
			continue
		}
		versions = append(versions, k)
	}
	sort.Slice(versions, func(i, j int) bool {
		return zigVersionGT(versions[i], versions[j])
	})
	if len(versions) == 0 {
		return "", "", "", "", fmt.Errorf("no stable versions found")
	}
	latest := versions[0]
	var entry map[string]json.RawMessage
	if err := json.Unmarshal(index[latest], &entry); err != nil {
		return "", "", "", "", err
	}
	var asset struct {
		Tarball string `json:"tarball"`
	}
	raw, ok := entry["x86_64-windows"]
	if !ok {
		return "", "", "", "", fmt.Errorf("no x86_64-windows asset for %s", latest)
	}
	if err := json.Unmarshal(raw, &asset); err != nil {
		return "", "", "", "", err
	}
	dlURL := asset.Tarball
	parts := strings.Split(dlURL, "/")
	file := parts[len(parts)-1]
	strip := strings.TrimSuffix(file, ".zip") + "/"
	log(fmt.Sprintf("  latest Zig stable: %s", latest))
	return dlURL, file, strip, latest, nil
}

func zigVersionGT(a, b string) bool {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var na, nb int
		if i < len(pa) {
			fmt.Sscanf(pa[i], "%d", &na)
		}
		if i < len(pb) {
			fmt.Sscanf(pb[i], "%d", &nb)
		}
		if na != nb {
			return na > nb
		}
	}
	return false
}

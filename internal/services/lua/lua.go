//go:build windows

// Package lua registers Lua with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package lua

import (
	"fmt"
	"regexp"

	"goampp/internal/packages"
)

func init() {
	packages.Register("Lua", packages.DownloadSpec{
		Version:    "5.4.8",
		URL:        "https://downloads.sourceforge.net/project/luabinaries/5.4.8/Tools%20Executables/lua-5.4.8_Win64_bin.zip",
		FileName:   "lua-5.4.8_Win64_bin.zip",
		InstallDir: "bin/lua",
		StripTop:   "",
		Kind:       packages.Zip,
		CheckFile:  "lua54.exe",
		Notes:      "Lua 5.4 scripting language. Executable is lua54.exe.",
	}, resolveLua54Latest)
}

func resolveLua54Latest(log func(string)) (urlOut, fileName, stripTop, version string, err error) {
	const pageURL = "https://sourceforge.net/projects/luabinaries/files/"
	log("  resolving latest LuaBinaries 5.4 Windows x64 build ...")
	body, err := packages.FetchReleaseText(pageURL)
	if err != nil {
		return "", "", "", "", err
	}
	re := regexp.MustCompile(`\b(5\.4\.[0-9]+)\b`)
	latest := ""
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		if len(m) == 2 && (latest == "" || packages.NumericVersionGT(m[1], latest)) {
			latest = m[1]
		}
	}
	if latest == "" {
		return "", "", "", "", fmt.Errorf("LuaBinaries 5.4 version not found")
	}
	file := "lua-" + latest + "_Win64_bin.zip"
	dlURL := fmt.Sprintf("https://downloads.sourceforge.net/project/luabinaries/%s/Tools%%20Executables/%s", latest, file)
	return dlURL, file, "", latest, nil
}

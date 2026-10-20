//go:build windows

// Package nodejs registers Node.js with the download catalogue:
// where to get it, how to find the newest build and how to set it up.
package nodejs

import "goampp/internal/packages"

func init() {
	packages.Register("Node.js", packages.DownloadSpec{
		Version:    "22.22.2 LTS",
		URL:        "https://nodejs.org/dist/v22.22.2/node-v22.22.2-win-x64.zip",
		FileName:   "node-v22.22.2-win-x64.zip",
		InstallDir: "bin/node",
		StripTop:   "node-v22.22.2-win-x64/",
		Kind:       packages.Zip,
		CheckFile:  "node.exe",
		Notes:      "JavaScript runtime — npm and npx are included alongside node.exe.",
		Variants: []packages.VariantSpec{
			{Version: "18", URL: "https://nodejs.org/dist/v18.20.5/node-v18.20.5-win-x64.zip", FileName: "node-v18.20.5-win-x64.zip", StripTop: "node-v18.20.5-win-x64/", Notes: "Node 18 LTS (Hydrogen, EOL April 2025)."},
			{Version: "20", URL: "https://nodejs.org/dist/v20.18.1/node-v20.18.1-win-x64.zip", FileName: "node-v20.18.1-win-x64.zip", StripTop: "node-v20.18.1-win-x64/", Notes: "Node 20 LTS (Iron)."},
			{Version: "22", URL: "https://nodejs.org/dist/v22.22.2/node-v22.22.2-win-x64.zip", FileName: "node-v22.22.2-win-x64.zip", StripTop: "node-v22.22.2-win-x64/", Notes: "Node 22 LTS (Jod) — current default."},
		},
	}, nil)
}

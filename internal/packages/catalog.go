//go:build windows

// Package packages is the download engine: the service catalogue that
// internal/services fills, latest-version resolvers, URL fallbacks, the
// download cache, extraction and installed-package metadata.
package packages

// FileKind selects how a downloaded artifact is installed. These were bare
// strings compared in a switch, so a typo in the catalogue fell through to the
// "unknown kind" arm at install time rather than failing to compile.
type FileKind string

const (
	Zip  FileKind = "zip"  // archive — extracted, with StripTop stripped off
	Exe  FileKind = "exe"  // silent installer — run with /S /D=<dir>
	File FileKind = "file" // one file — copied to TargetFile
)

type DownloadSpec struct {
	Version    string
	URL        string
	FileName   string
	InstallDir string

	StripTop string

	Kind FileKind

	TargetFile string

	CheckFile string

	PostInstall func(installDir string, log func(string)) error

	Notes string

	Variants []VariantSpec

	URLResolver func(log func(string)) (url, fileName, stripTop, version string, err error)
}

type VariantSpec struct {
	Version  string
	URL      string
	FileName string
	StripTop string

	Notes string
}

var DownloadCatalog = map[string]DownloadSpec{}

// Register adds a service's download recipe to the catalogue. Each service
// package calls it from init. resolver, when non-nil, finds the newest build
// at install time; the spec's own URL stays as the offline fallback.
func Register(name string, spec DownloadSpec, resolver LatestResolver) {
	DownloadCatalog[name] = spec
	if resolver != nil {
		registerLatestResolver(name, resolver)
	}
}

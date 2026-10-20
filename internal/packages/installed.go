//go:build windows

package packages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type InstalledPackage struct {
	Name        string    `json:"name"`
	Variant     string    `json:"variant,omitempty"`
	Version     string    `json:"version"`
	PackageID   string    `json:"package_id"`
	FileName    string    `json:"file_name"`
	URL         string    `json:"url,omitempty"`
	InstalledAt time.Time `json:"installed_at"`
	Legacy      bool      `json:"legacy,omitempty"`
}

type ServiceUpdateInfo struct {
	Name      string
	Installed bool
	Current   InstalledPackage
	Latest    DownloadResolution
	Available bool
	Err       error
}

var (
	packageMetaMu     sync.Mutex
	UpdateOperationMu sync.Mutex
	UpdateStateMu     sync.RWMutex
	ServiceUpdates    = map[string]ServiceUpdateInfo{}
)

var packageSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

func packageMetadataPath(baseDir, name string) string {
	slug := strings.Trim(packageSlugRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
	return filepath.Join(baseDir, "data", ".goampp", "packages", slug+".json")
}

func WriteInstalledPackage(baseDir, name, variant string, resolved DownloadResolution) error {
	packageMetaMu.Lock()
	defer packageMetaMu.Unlock()

	meta := InstalledPackage{
		Name:        name,
		Variant:     variant,
		Version:     resolved.Version,
		PackageID:   resolved.PackageID,
		FileName:    resolved.FileName,
		URL:         resolved.URL,
		InstalledAt: time.Now(),
	}

	path := packageMetadataPath(baseDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readInstalledPackage(baseDir, name, variant string) (InstalledPackage, bool) {
	if !IsInstalled(name, baseDir) {
		return InstalledPackage{}, false
	}

	// These services expose their exact installed version themselves. Prefer
	// that over metadata/catalogue guesses, particularly after branch aliases
	// such as PHP 8.4 or Node 22 have moved to a newer patch release.
	if IsDynamicVariantService(name) {
		if version, ok := detectInstalledRuntimeVersion(baseDir, name); ok {
			return InstalledPackage{
				Name:      name,
				Variant:   EffectiveVariant(name, variant),
				Version:   version,
				PackageID: PackageIdentity(version, ""),
				Legacy:    true,
			}, true
		}
	}

	path := packageMetadataPath(baseDir, name)
	data, err := os.ReadFile(path)
	if err == nil {
		var meta InstalledPackage
		if json.Unmarshal(data, &meta) == nil && meta.Name != "" {
			return meta, true
		}
	}

	// Migration path for installs created before package metadata existed.
	// GoAMPP historically installed exactly the package named in the catalogue,
	// so use that as a best-effort baseline until the component is reinstalled.
	spec, ok := DownloadCatalog[name]
	if !ok {
		return InstalledPackage{}, false
	}

	meta := InstalledPackage{
		Name:      name,
		Variant:   variant,
		Version:   spec.Version,
		PackageID: PackageIdentity(spec.Version, spec.FileName),
		FileName:  spec.FileName,
		URL:       spec.URL,
		Legacy:    true,
	}
	if len(spec.Variants) > 0 && variant != "" {
		for _, v := range spec.Variants {
			if v.Version != variant {
				continue
			}
			meta.Version = v.Version
			meta.PackageID = PackageIdentity(v.Version, v.FileName)
			meta.FileName = v.FileName
			meta.URL = v.URL
			break
		}
	}
	return meta, true
}

func CheckServiceUpdate(name, variant, baseDir string, log func(string)) ServiceUpdateInfo {
	info := ServiceUpdateInfo{Name: name}

	// Resolve the upstream package even when the component is not installed.
	// The Services page uses this same result for its visible version labels.
	var (
		latest DownloadResolution
		err    error
	)
	if IsDynamicVariantService(name) {
		var ok bool
		latest, ok = ResolvedVariantDownload(name, variant)
		if !ok {
			err = fmt.Errorf("latest %s branch package not resolved", EffectiveVariant(name, variant))
		}
	} else {
		latest, err = resolveLatestDownload(name, variant, log)
	}
	if err != nil {
		info.Err = err
		return info
	}
	info.Latest = latest

	current, installed := readInstalledPackage(baseDir, name, variant)
	if !installed {
		return info
	}
	info.Installed = true
	info.Current = current

	if IsDynamicVariantService(name) {
		info.Available = strings.TrimSpace(current.Version) != strings.TrimSpace(latest.Version)
		return info
	}

	if current.PackageID == "" || latest.PackageID == "" {
		info.Err = fmt.Errorf("package identity unavailable")
		return info
	}
	info.Available = current.PackageID != latest.PackageID
	return info
}

func CachedServiceUpdate(name string) (ServiceUpdateInfo, bool) {
	UpdateStateMu.RLock()
	defer UpdateStateMu.RUnlock()
	info, ok := ServiceUpdates[name]
	return info, ok
}

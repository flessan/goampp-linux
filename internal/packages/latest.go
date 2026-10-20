//go:build windows

package packages

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"goampp/internal/env"
)

type DownloadResolution struct {
	Version   string
	PackageID string
	URL       string
	FileName  string
	StripTop  string
}

type LatestResolver func(log func(string)) (url, fileName, stripTop, version string, err error)

type latestCacheEntry struct {
	URL       string
	FileName  string
	StripTop  string
	Version   string
	CheckedAt time.Time
}

var (
	latestCacheMu sync.Mutex
	latestCache   = map[string]latestCacheEntry{}

	InstallResolveMu     sync.Mutex
	ResolvedFreshInstall = map[string]bool{}
)

const latestCacheTTL = 5 * time.Minute

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func PackageIdentity(version, fileName string) string {
	version = strings.TrimSpace(version)
	fileName = strings.TrimSpace(fileName)
	if version == "" {
		return fileName
	}
	if fileName == "" {
		return version
	}
	return version + "|" + fileName
}

func cachedLatestResolver(key string, resolver LatestResolver) LatestResolver {
	return func(log func(string)) (url, fileName, stripTop, version string, err error) {
		latestCacheMu.Lock()
		entry, ok := latestCache[key]
		if ok && time.Since(entry.CheckedAt) < latestCacheTTL {
			latestCacheMu.Unlock()
			return entry.URL, entry.FileName, entry.StripTop, entry.Version, nil
		}
		latestCacheMu.Unlock()

		url, fileName, stripTop, version, err = resolver(log)
		if err != nil {
			return "", "", "", "", err
		}
		latestCacheMu.Lock()
		latestCache[key] = latestCacheEntry{
			URL: url, FileName: fileName, StripTop: stripTop,
			Version: version, CheckedAt: time.Now(),
		}
		latestCacheMu.Unlock()
		return url, fileName, stripTop, version, nil
	}
}

func ResolveGitHubLatestAsset(owner, repo string, assetPattern *regexp.Regexp) LatestResolver {
	return func(log func(string)) (url, fileName, stripTop, version string, err error) {
		log(fmt.Sprintf("  resolving latest %s/%s release ...", owner, repo))
		release, err := GithubLatestRelease(owner, repo, log)
		if err != nil {
			return "", "", "", "", err
		}
		for _, asset := range release.Assets {
			if !assetPattern.MatchString(asset.Name) {
				continue
			}
			resolvedVersion := strings.TrimPrefix(release.TagName, "v")
			log(fmt.Sprintf("  latest %s/%s release: %s", owner, repo, resolvedVersion))
			return asset.BrowserDownloadURL, asset.Name, "", resolvedVersion, nil
		}
		return "", "", "", "", fmt.Errorf("no matching asset in %s/%s release %s", owner, repo, release.TagName)
	}
}

func ResolveGitHubSeriesAsset(owner, repo, tagPrefix string, assetPattern *regexp.Regexp) LatestResolver {
	return func(log func(string)) (url, fileName, stripTop, version string, err error) {
		log(fmt.Sprintf("  resolving latest %s/%s release in %s series ...", owner, repo, tagPrefix))
		releases, err := githubRecentReleases(owner, repo, log)
		if err != nil {
			return "", "", "", "", err
		}
		for _, release := range releases {
			if release.Draft || release.Prerelease || !strings.HasPrefix(release.TagName, tagPrefix) {
				continue
			}
			release = githubReleaseWithAssets(owner, repo, release)
			for _, asset := range release.Assets {
				if !assetPattern.MatchString(asset.Name) {
					continue
				}
				resolvedVersion := strings.TrimPrefix(release.TagName, "v")
				return asset.BrowserDownloadURL, asset.Name, "", resolvedVersion, nil
			}
		}
		return "", "", "", "", fmt.Errorf("no matching %s/%s release in series %s", owner, repo, tagPrefix)
	}
}

func WithStripTop(resolver LatestResolver, strip func(fileName, version string) string) LatestResolver {
	return func(log func(string)) (url, fileName, stripTop, version string, err error) {
		url, fileName, _, version, err = resolver(log)
		if err != nil {
			return "", "", "", "", err
		}
		return url, fileName, strip(fileName, version), version, nil
	}
}

func resolveLatestDownload(name, variant string, log func(string)) (DownloadResolution, error) {
	spec, ok := DownloadCatalog[name]
	if !ok {
		return DownloadResolution{}, fmt.Errorf("no download info registered for %q", name)
	}

	resolved := DownloadResolution{
		Version:   spec.Version,
		PackageID: PackageIdentity(spec.Version, spec.FileName),
		URL:       spec.URL,
		FileName:  spec.FileName,
		StripTop:  spec.StripTop,
	}

	if len(spec.Variants) > 0 && variant != "" {
		for _, v := range spec.Variants {
			if v.Version != variant {
				continue
			}
			return DownloadResolution{
				Version:   v.Version,
				PackageID: PackageIdentity(v.Version, v.FileName),
				URL:       v.URL,
				FileName:  v.FileName,
				StripTop:  v.StripTop,
			}, nil
		}
		return DownloadResolution{}, fmt.Errorf("[%s] version %q not in catalogue", name, variant)
	}

	if spec.URLResolver == nil {
		return resolved, nil
	}

	url, fileName, stripTop, version, err := spec.URLResolver(log)
	if err != nil {
		return DownloadResolution{}, err
	}
	if fileName == "" {
		fileName = spec.FileName
	}
	if version == "" {
		version = spec.Version
	}
	return DownloadResolution{
		Version:   version,
		PackageID: PackageIdentity(version, fileName),
		URL:       url,
		FileName:  fileName,
		StripTop:  stripTop,
	}, nil
}

func registerLatestResolver(name string, resolver LatestResolver) {
	spec, ok := DownloadCatalog[name]
	if !ok {
		return
	}

	cachedResolver := cachedLatestResolver(name, resolver)
	spec.URLResolver = func(log func(string)) (url, fileName, stripTop, version string, err error) {
		if env.BaseDir != "" {
			InstallResolveMu.Lock()
			ResolvedFreshInstall[name] = !IsInstalled(name, env.BaseDir)
			InstallResolveMu.Unlock()
		}
		return cachedResolver(log)
	}

	originalPostInstall := spec.PostInstall
	installDirParts := strings.FieldsFunc(filepath.ToSlash(spec.InstallDir), func(r rune) bool { return r == '/' })

	spec.PostInstall = func(installDir string, log func(string)) error {
		if originalPostInstall != nil {
			if err := originalPostInstall(installDir, log); err != nil {
				return err
			}
		}

		InstallResolveMu.Lock()
		freshInstall := ResolvedFreshInstall[name]
		delete(ResolvedFreshInstall, name)
		InstallResolveMu.Unlock()
		if !freshInstall {
			return nil
		}

		// Record what was actually installed: when the resolved package failed
		// to download, the install fell back to the pinned or archive URL.
		var url, fileName, stripTop, version string
		if used, ok := takeInstalledCandidate(name); ok {
			url, fileName, stripTop, version = used.URL, used.FileName, used.StripTop, used.Version
		} else {
			var err error
			if url, fileName, stripTop, version, err = cachedResolver(log); err != nil {
				log(fmt.Sprintf("  package metadata skipped: %v", err))
				return nil
			}
		}

		baseDir := installDir
		for range installDirParts {
			baseDir = filepath.Dir(baseDir)
		}
		resolved := DownloadResolution{
			Version:   version,
			PackageID: PackageIdentity(version, fileName),
			URL:       url,
			FileName:  fileName,
			StripTop:  stripTop,
		}
		if err := WriteInstalledPackage(baseDir, name, "", resolved); err != nil {
			log(fmt.Sprintf("  package metadata write failed: %v", err))
		} else {
			log(fmt.Sprintf("  recorded installed package %s", fileName))
		}
		return nil
	}

	DownloadCatalog[name] = spec
}

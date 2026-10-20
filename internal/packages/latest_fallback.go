//go:build windows

package packages

import (
	"fmt"
	"strings"
	"sync"
)

// downloadCandidate is one place a package can be fetched from. Install tries
// candidates in order — resolved latest, then the pinned catalogue URL, then
// known archive locations of both — and moves on whenever one fails (404, 410,
// an HTML error page under a 200, ...). Each candidate carries its own file
// name and strip prefix because they differ between versions.
type downloadCandidate struct {
	Source   string
	URL      string
	FileName string
	StripTop string
	Version  string
}

var (
	installedCandidateMu sync.Mutex
	installedCandidates  = map[string]downloadCandidate{}
)

// archiveFallbackURLs returns where a pinned file usually lives once upstream
// retires it from the primary location.
func archiveFallbackURLs(rawURL string) []string {
	var out []string
	for _, prefix := range []string{
		"https://windows.php.net/downloads/releases/",
		"https://downloads.php.net/~windows/releases/",
	} {
		rest, ok := strings.CutPrefix(rawURL, prefix)
		if !ok {
			continue
		}
		file := strings.TrimPrefix(rest, "archives/")
		out = append(out,
			"https://downloads.php.net/~windows/releases/archives/"+file,
			"https://windows.php.net/downloads/releases/archives/"+file,
		)
	}
	// SourceForge mirror hosts come and go; the redirector always picks a live one.
	if i := strings.Index(rawURL, ".dl.sourceforge.net/project/"); i > 0 {
		out = append(out, "https://downloads.sourceforge.net"+rawURL[i+len(".dl.sourceforge.net"):])
	}
	// Lightbend stopped serving Scala; the same artifacts are on GitHub.
	if rest, ok := strings.CutPrefix(rawURL, "https://downloads.lightbend.com/scala/"); ok {
		if ver, file, ok := strings.Cut(rest, "/"); ok {
			out = append(out, "https://github.com/scala/scala/releases/download/v"+ver+"/"+file)
		}
	}
	return out
}

// downloadCandidates builds the ordered fallback list for an install.
func DownloadCandidates(name, variant string, log func(string)) ([]downloadCandidate, error) {
	spec, ok := DownloadCatalog[name]
	if !ok {
		return nil, fmt.Errorf("no download info registered for %q", name)
	}

	var out []downloadCandidate
	add := func(c downloadCandidate) {
		if c.URL == "" {
			return
		}
		for _, have := range out {
			if have.URL == c.URL {
				return
			}
		}
		out = append(out, c)
	}

	pinned := downloadCandidate{Source: "pinned", URL: spec.URL, FileName: spec.FileName, StripTop: spec.StripTop, Version: spec.Version}
	if len(spec.Variants) > 0 && variant != "" {
		found := false
		for _, v := range spec.Variants {
			if v.Version == variant {
				pinned = downloadCandidate{Source: "pinned", URL: v.URL, FileName: v.FileName, StripTop: v.StripTop, Version: v.Version}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("[%s] version %q not in catalogue", name, variant)
		}
	}

	switch {
	case IsDynamicVariantService(name):
		// Branch services (PHP 8.4, Node 22, ...) resolve the newest patch
		// release of the selected — or default — branch at install time.
		branch := EffectiveVariant(name, variant)
		if r, err := resolveVariantNow(name, branch, log); err != nil {
			log(fmt.Sprintf("[%s] %s release resolve failed (%v), falling back to bundled URL", name, branch, err))
		} else {
			add(downloadCandidate{Source: "latest", URL: r.URL, FileName: r.FileName, StripTop: r.StripTop, Version: r.Version})
		}
	case spec.URLResolver != nil && variant == "":
		if u, f, s, v, err := spec.URLResolver(log); err != nil {
			log(fmt.Sprintf("[%s] version resolve failed (%v), falling back to bundled URL", name, err))
		} else {
			if f == "" {
				f = spec.FileName
			}
			add(downloadCandidate{Source: "latest", URL: u, FileName: f, StripTop: s, Version: v})
		}
	}

	add(pinned)
	for _, c := range append([]downloadCandidate(nil), out...) {
		for _, alt := range archiveFallbackURLs(c.URL) {
			a := c
			a.URL, a.Source = alt, c.Source+" (archive)"
			add(a)
		}
	}
	return out, nil
}

// fetchFirstCandidate downloads the first candidate that works.
func FetchFirstCandidate(cache *downloadCache, name string, candidates []downloadCandidate, log func(string), onProgress func(done, total int64)) (string, downloadCandidate, error) {
	var errs []string
	for i, c := range candidates {
		log(fmt.Sprintf("  source: %s (%s)", c.Source, c.Version))
		path, err := cache.Fetch(c.FileName, c.URL, onProgress)
		if err == nil {
			return path, c, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", c.URL, err))
		if i < len(candidates)-1 {
			log(fmt.Sprintf("[%s] %s download failed (%v), trying next source", name, c.Source, err))
		}
	}
	if len(errs) == 0 {
		return "", downloadCandidate{}, fmt.Errorf("no download URL")
	}
	return "", downloadCandidate{}, fmt.Errorf("all sources failed: %s", strings.Join(errs, "; "))
}

func recordInstalledCandidate(name string, c downloadCandidate) {
	installedCandidateMu.Lock()
	installedCandidates[name] = c
	installedCandidateMu.Unlock()
}

func takeInstalledCandidate(name string) (downloadCandidate, bool) {
	installedCandidateMu.Lock()
	defer installedCandidateMu.Unlock()
	c, ok := installedCandidates[name]
	delete(installedCandidates, name)
	return c, ok
}

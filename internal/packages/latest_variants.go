//go:build windows

package packages

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"

	"goampp/internal/env"
)

type variantKey struct {
	Name    string
	Variant string
}

var (
	variantRefreshMu sync.Mutex
	variantStateMu   sync.RWMutex
	latestVariants   = map[variantKey]DownloadResolution{}
)

func EffectiveVariant(name, selected string) string {
	if selected != "" {
		return selected
	}
	spec, ok := DownloadCatalog[name]
	if !ok || len(spec.Variants) == 0 {
		return ""
	}
	for _, v := range spec.Variants {
		if strings.HasPrefix(spec.Version, v.Version) {
			return v.Version
		}
	}
	return spec.Variants[len(spec.Variants)-1].Version
}

func ResolvedVariantDownload(name, selected string) (DownloadResolution, bool) {
	variant := EffectiveVariant(name, selected)
	if variant == "" {
		return DownloadResolution{}, false
	}
	variantStateMu.RLock()
	defer variantStateMu.RUnlock()
	resolved, ok := latestVariants[variantKey{Name: name, Variant: variant}]
	return resolved, ok
}

var variantResolvers = map[string]func(func(string)) (map[string]DownloadResolution, error){
	"PHP-FPM": resolvePHPVariants,
	"Node.js": resolveNodeVariants,
	"Python":  resolvePythonVariants,
}

// resolveVariantNow returns the newest package of one branch, using the last
// refresh when there was one and asking upstream otherwise — so an install
// never depends on the background refresh having succeeded first.
func resolveVariantNow(name, variant string, log func(string)) (DownloadResolution, error) {
	if resolved, ok := ResolvedVariantDownload(name, variant); ok {
		return resolved, nil
	}
	fn, ok := variantResolvers[name]
	if !ok {
		return DownloadResolution{}, fmt.Errorf("%s has no branch resolver", name)
	}
	all, err := fn(log)
	if err != nil {
		return DownloadResolution{}, err
	}
	variantStateMu.Lock()
	for v, r := range all {
		latestVariants[variantKey{Name: name, Variant: v}] = r
	}
	variantStateMu.Unlock()
	resolved, ok := all[variant]
	if !ok {
		return DownloadResolution{}, fmt.Errorf("%s %s not found upstream", name, variant)
	}
	return resolved, nil
}

func RefreshVariantCatalog(verbose bool) error {
	variantRefreshMu.Lock()
	defer variantRefreshMu.Unlock()

	log := func(string) {}
	if verbose && env.BaseDir != "" {
		log = env.Log
	}

	type result struct {
		name     string
		resolved map[string]DownloadResolution
		err      error
	}
	results := make(chan result, len(variantResolvers))
	var wg sync.WaitGroup

	for name, fn := range variantResolvers {
		name, fn := name, fn
		wg.Add(1)
		go func() {
			defer wg.Done()
			resolved, err := fn(log)
			results <- result{name: name, resolved: resolved, err: err}
		}()
	}
	wg.Wait()
	close(results)

	fresh := map[variantKey]DownloadResolution{}
	var errs []string
	for r := range results {
		if r.err != nil {
			errs = append(errs, r.name+": "+r.err.Error())
			continue
		}
		for variant, resolved := range r.resolved {
			fresh[variantKey{Name: r.name, Variant: variant}] = resolved
		}
	}

	variantStateMu.Lock()
	for key, resolved := range fresh {
		latestVariants[key] = resolved
	}
	variantStateMu.Unlock()

	// Keep the branch selector itself stable (8.4, 22, 3.13), but replace
	// its backing package so the existing installer automatically downloads
	// the newest patch release on the next install/version switch. Build a
	// complete copy instead of mutating the shared map/slices in place: UI
	// readers can safely finish using the previous immutable snapshot.
	nextCatalog := make(map[string]DownloadSpec, len(DownloadCatalog))
	for name, spec := range DownloadCatalog {
		if len(spec.Variants) > 0 {
			spec.Variants = append([]VariantSpec(nil), spec.Variants...)
		}
		nextCatalog[name] = spec
	}
	for _, serviceName := range []string{"PHP-FPM", "Node.js", "Python"} {
		spec, ok := nextCatalog[serviceName]
		if !ok {
			continue
		}
		changed := false
		for i := range spec.Variants {
			key := variantKey{Name: serviceName, Variant: spec.Variants[i].Version}
			resolved, ok := fresh[key]
			if !ok {
				continue
			}
			spec.Variants[i].URL = resolved.URL
			spec.Variants[i].FileName = resolved.FileName
			spec.Variants[i].StripTop = resolved.StripTop
			changed = true
		}
		if changed {
			nextCatalog[serviceName] = spec
		}
	}
	DownloadCatalog = nextCatalog

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func resolveNodeVariants(log func(string)) (map[string]DownloadResolution, error) {
	const apiURL = "https://nodejs.org/dist/index.json"
	log("  resolving Node.js branch releases ...")
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GoAMPP")
	resp, err := LatestHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Node.js index returned HTTP %d", resp.StatusCode)
	}

	var releases []struct {
		Version string   `json:"version"`
		Files   []string `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	wanted := map[string]bool{}
	if spec, ok := DownloadCatalog["Node.js"]; ok {
		for _, v := range spec.Variants {
			wanted[v.Version] = true
		}
	}
	out := map[string]DownloadResolution{}
	for _, release := range releases {
		v := strings.TrimPrefix(release.Version, "v")
		parts := strings.Split(v, ".")
		if len(parts) < 3 || !wanted[parts[0]] {
			continue
		}
		hasWindowsZip := false
		for _, fileKind := range release.Files {
			if fileKind == "win-x64-zip" {
				hasWindowsZip = true
				break
			}
		}
		if !hasWindowsZip {
			continue
		}
		branch := parts[0]
		if _, exists := out[branch]; exists {
			continue
		}
		file := "node-v" + v + "-win-x64.zip"
		out[branch] = DownloadResolution{
			Version:   v,
			PackageID: PackageIdentity(v, file),
			URL:       "https://nodejs.org/dist/v" + v + "/" + file,
			FileName:  file,
			StripTop:  strings.TrimSuffix(file, ".zip") + "/",
		}
	}
	for branch := range wanted {
		if _, ok := out[branch]; !ok {
			return nil, fmt.Errorf("Node.js %s Windows x64 ZIP not found", branch)
		}
	}
	return out, nil
}

func phpCompilerKey(branch string) (string, bool) {
	switch branch {
	case "7.4":
		return "nts-vc15-x64", true
	case "8.0", "8.1", "8.2", "8.3":
		return "nts-vs16-x64", true
	case "8.4", "8.5":
		return "nts-vs17-x64", true
	default:
		return "", false
	}
}

func resolvePHPVariants(log func(string)) (map[string]DownloadResolution, error) {
	const apiURL = "https://downloads.php.net/~windows/releases/releases.json"
	log("  resolving PHP Windows branch releases ...")
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GoAMPP")
	resp, err := LatestHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("PHP releases index returned HTTP %d", resp.StatusCode)
	}

	var releases map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	out := map[string]DownloadResolution{}
	spec, ok := DownloadCatalog["PHP-FPM"]
	if !ok {
		return out, nil
	}
	// One retired or missing branch must not take the others down with it.
	resolveBranch := func(branch, compilerKey string) (DownloadResolution, error) {
		raw, ok := releases[branch]
		if !ok {
			return DownloadResolution{}, fmt.Errorf("PHP %s missing from releases index", branch)
		}
		var entry map[string]json.RawMessage
		if err := json.Unmarshal(raw, &entry); err != nil {
			return DownloadResolution{}, err
		}
		var version string
		if err := json.Unmarshal(entry["version"], &version); err != nil || version == "" {
			return DownloadResolution{}, fmt.Errorf("PHP %s exact version missing", branch)
		}
		var asset struct {
			Zip struct {
				Path string `json:"path"`
			} `json:"zip"`
		}
		if err := json.Unmarshal(entry[compilerKey], &asset); err != nil {
			return DownloadResolution{}, fmt.Errorf("PHP %s %s metadata: %w", branch, compilerKey, err)
		}
		file := asset.Zip.Path
		if file == "" {
			file = fmt.Sprintf("php-%s-nts-Win32-%s.zip", version, strings.TrimPrefix(compilerKey, "nts-"))
		}

		var downloadURL string
		for _, baseURL := range []string{
			"https://downloads.php.net/~windows/releases/",
			"https://downloads.php.net/~windows/releases/archives/",
		} {
			candidate := baseURL + file
			if urlExists(candidate) {
				downloadURL = candidate
				break
			}
		}
		if downloadURL == "" {
			return DownloadResolution{}, fmt.Errorf("PHP %s exact Windows archive %s not found", branch, file)
		}

		return DownloadResolution{
			Version:   version,
			PackageID: PackageIdentity(version, file),
			URL:       downloadURL,
			FileName:  file,
			StripTop:  "",
		}, nil
	}

	var errs []string
	for _, variant := range spec.Variants {
		compilerKey, ok := phpCompilerKey(variant.Version)
		if !ok {
			continue
		}
		resolved, err := resolveBranch(variant.Version, compilerKey)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		out[variant.Version] = resolved
	}
	if len(out) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return out, nil
}

func urlExists(rawURL string) bool {
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err == nil {
		req.Header.Set("User-Agent", "GoAMPP")
		if resp, err := LatestHTTPClient.Do(req); err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return true
			}
		}
	}

	// Some mirrors/CDNs reject HEAD. A one-byte ranged GET is the fallback.
	req, err = http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "GoAMPP")
	req.Header.Set("Range", "bytes=0-0")
	resp, err := LatestHTTPClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusPartialContent || (resp.StatusCode >= 200 && resp.StatusCode < 300)
}

func resolvePythonVariants(log func(string)) (map[string]DownloadResolution, error) {
	const indexURL = "https://www.python.org/ftp/python/"
	log("  resolving Python embeddable branch releases ...")
	body, err := FetchReleaseText(indexURL)
	if err != nil {
		return nil, err
	}

	spec, ok := DownloadCatalog["Python"]
	if !ok {
		return map[string]DownloadResolution{}, nil
	}
	wanted := map[string]bool{}
	for _, v := range spec.Variants {
		wanted[v.Version] = true
	}

	versionRe := regexp.MustCompile(`href=["'](3\.[0-9]+\.[0-9]+)/["']`)
	byBranch := map[string][]string{}
	for _, m := range versionRe.FindAllStringSubmatch(body, -1) {
		if len(m) != 2 {
			continue
		}
		parts := strings.Split(m[1], ".")
		if len(parts) != 3 {
			continue
		}
		branch := parts[0] + "." + parts[1]
		if wanted[branch] {
			byBranch[branch] = append(byBranch[branch], m[1])
		}
	}

	out := map[string]DownloadResolution{}
	for branch := range wanted {
		versions := byBranch[branch]
		sort.Slice(versions, func(i, j int) bool { return NumericVersionGT(versions[i], versions[j]) })
		for _, version := range versions {
			candidates := []string{
				fmt.Sprintf("python-%s-embeddable-amd64.zip", version),
				fmt.Sprintf("python-%s-embed-amd64.zip", version),
			}
			found := ""
			for _, file := range candidates {
				dlURL := indexURL + version + "/" + file
				if urlExists(dlURL) {
					found = file
					break
				}
			}
			if found == "" {
				continue
			}
			out[branch] = DownloadResolution{
				Version:   version,
				PackageID: PackageIdentity(version, found),
				URL:       indexURL + version + "/" + found,
				FileName:  found,
				StripTop:  "",
			}
			break
		}
		if _, ok := out[branch]; !ok {
			return nil, fmt.Errorf("Python %s embeddable amd64 archive not found", branch)
		}
	}
	return out, nil
}

var detectedVersionPatterns = map[string]*regexp.Regexp{
	"PHP-FPM": regexp.MustCompile(`(?i)PHP\s+([0-9]+\.[0-9]+\.[0-9]+)`),
	"Node.js": regexp.MustCompile(`(?i)^v?([0-9]+\.[0-9]+\.[0-9]+)`),
	"Python":  regexp.MustCompile(`(?i)Python\s+([0-9]+\.[0-9]+\.[0-9]+)`),
}

func detectInstalledRuntimeVersion(baseDir, name string) (string, bool) {
	var exe string
	var args []string
	switch name {
	case "PHP-FPM":
		exe = baseDir + `\bin\php\php-cgi.exe`
		args = []string{"-v"}
	case "Node.js":
		exe = baseDir + `\bin\node\node.exe`
		args = []string{"--version"}
	case "Python":
		exe = baseDir + `\bin\python\python.exe`
		args = []string{"--version"}
	default:
		return "", false
	}
	out, err := exec.Command(exe, args...).CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", false
	}
	re := detectedVersionPatterns[name]
	m := re.FindStringSubmatch(strings.TrimSpace(string(out)))
	if len(m) != 2 {
		return "", false
	}
	return m[1], true
}

func IsDynamicVariantService(name string) bool {
	switch name {
	case "PHP-FPM", "Node.js", "Python":
		return true
	default:
		return false
	}
}

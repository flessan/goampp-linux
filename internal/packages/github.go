//go:build windows

package packages

// GitHub release lookups. The REST API allows only 60 unauthenticated calls
// an hour per IP, which a single "Check updates" pass can exhaust, so every
// lookup goes API -> github.com web pages (not rate limited) and results are
// cached for the life of the process. GITHUB_TOKEN / GH_TOKEN, when set, lift
// the API limit to 5000 calls.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ghCacheMu      sync.Mutex
	ghLatestCache  = map[string]githubRelease{}
	ghListCache    = map[string][]githubRelease{}
	ghAssetsCache  = map[string]githubRelease{}
	ghHTTP         = &http.Client{Timeout: 20 * time.Second}
	ghNoRedirect   = &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	ghAssetHref    = regexp.MustCompile(`href="(/[^"]+/releases/download/[^"]+)"`)
	ghAtomTag      = regexp.MustCompile(`/releases/tag/([^"<>]+)"`)
	ghPrereleaseRe = regexp.MustCompile(`(?i)(rc|alpha|beta|preview|pre)[.\-]?\d*$`)
)

func githubToken() string {
	for _, k := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func githubAPIGet(url string, out any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "GoAMPP")
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := githubToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := ghHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func githubWebGet(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoAMPP")
	resp, err := ghHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github.com returned HTTP %d for %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	return string(b), err
}

// githubReleaseAssetsFromWeb fills in the asset list of one tag from the
// release page's expanded_assets fragment.
func githubReleaseAssetsFromWeb(owner, repo, tag string) (githubRelease, error) {
	key := owner + "/" + repo + "@" + tag
	ghCacheMu.Lock()
	if r, ok := ghAssetsCache[key]; ok {
		ghCacheMu.Unlock()
		return r, nil
	}
	ghCacheMu.Unlock()

	html, err := githubWebGet(fmt.Sprintf("https://github.com/%s/%s/releases/expanded_assets/%s", owner, repo, tag))
	if err != nil {
		return githubRelease{}, err
	}
	r := githubRelease{TagName: tag}
	seen := map[string]bool{}
	for _, m := range ghAssetHref.FindAllStringSubmatch(html, -1) {
		href := m[1]
		if seen[href] {
			continue
		}
		seen[href] = true
		name := href[strings.LastIndex(href, "/")+1:]
		r.Assets = append(r.Assets, struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{Name: name, BrowserDownloadURL: "https://github.com" + href})
	}
	ghCacheMu.Lock()
	ghAssetsCache[key] = r
	ghCacheMu.Unlock()
	return r, nil
}

// githubLatestRelease returns the latest non-prerelease release with assets.
func GithubLatestRelease(owner, repo string, log func(string)) (githubRelease, error) {
	key := owner + "/" + repo
	ghCacheMu.Lock()
	if r, ok := ghLatestCache[key]; ok {
		ghCacheMu.Unlock()
		return r, nil
	}
	ghCacheMu.Unlock()

	var r githubRelease
	apiErr := githubAPIGet(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo), &r)
	if apiErr != nil {
		log(fmt.Sprintf("  %s: %v — using github.com release page", key, apiErr))
		resp, err := ghNoRedirect.Get(fmt.Sprintf("https://github.com/%s/%s/releases/latest", owner, repo))
		if err != nil {
			return githubRelease{}, err
		}
		resp.Body.Close()
		loc := resp.Header.Get("Location")
		i := strings.Index(loc, "/releases/tag/")
		if i < 0 {
			return githubRelease{}, fmt.Errorf("%s: no latest release (API: %v)", key, apiErr)
		}
		tag := loc[i+len("/releases/tag/"):]
		if r, err = githubReleaseAssetsFromWeb(owner, repo, tag); err != nil {
			return githubRelease{}, err
		}
	}
	ghCacheMu.Lock()
	ghLatestCache[key] = r
	ghCacheMu.Unlock()
	return r, nil
}

// githubRecentReleases lists recent releases, newest first. On the web
// fallback only tags are known; assets are fetched per tag on demand by
// githubReleaseWithAssets.
func githubRecentReleases(owner, repo string, log func(string)) ([]githubRelease, error) {
	key := owner + "/" + repo
	ghCacheMu.Lock()
	if r, ok := ghListCache[key]; ok {
		ghCacheMu.Unlock()
		return r, nil
	}
	ghCacheMu.Unlock()

	var list []githubRelease
	apiErr := githubAPIGet(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=50", owner, repo), &list)
	if apiErr != nil {
		log(fmt.Sprintf("  %s: %v — using github.com release feed", key, apiErr))
		feed, err := githubWebGet(fmt.Sprintf("https://github.com/%s/%s/releases.atom", owner, repo))
		if err != nil {
			return nil, err
		}
		list = nil
		seen := map[string]bool{}
		for _, m := range ghAtomTag.FindAllStringSubmatch(feed, -1) {
			tag := m[1]
			if seen[tag] {
				continue
			}
			seen[tag] = true
			list = append(list, githubRelease{TagName: tag, Prerelease: ghPrereleaseRe.MatchString(tag)})
		}
	}
	ghCacheMu.Lock()
	ghListCache[key] = list
	ghCacheMu.Unlock()
	return list, nil
}

func githubReleaseWithAssets(owner, repo string, r githubRelease) githubRelease {
	if len(r.Assets) > 0 {
		return r
	}
	if full, err := githubReleaseAssetsFromWeb(owner, repo, r.TagName); err == nil {
		return full
	}
	return r
}

// resetGitHubCache drops cached lookups so an explicit update check sees
// releases published since the last one.
func ResetGitHubCache() {
	ghCacheMu.Lock()
	ghLatestCache = map[string]githubRelease{}
	ghListCache = map[string][]githubRelease{}
	ghCacheMu.Unlock()
}

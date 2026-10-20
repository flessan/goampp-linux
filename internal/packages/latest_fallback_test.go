//go:build windows

package packages

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestArchiveFallbackURLs(t *testing.T) {
	cases := map[string][]string{
		"https://windows.php.net/downloads/releases/php-8.4.22-nts-Win32-vs17-x64.zip": {
			"https://downloads.php.net/~windows/releases/archives/php-8.4.22-nts-Win32-vs17-x64.zip",
			"https://windows.php.net/downloads/releases/archives/php-8.4.22-nts-Win32-vs17-x64.zip",
		},
		"https://iweb.dl.sourceforge.net/project/luabinaries/5.4.7/x.zip": {
			"https://downloads.sourceforge.net/project/luabinaries/5.4.7/x.zip",
		},
		"https://downloads.lightbend.com/scala/2.13.16/scala-2.13.16.zip": {
			"https://github.com/scala/scala/releases/download/v2.13.16/scala-2.13.16.zip",
		},
		"https://nginx.org/download/nginx-1.28.3.zip": nil,
	}
	for in, want := range cases {
		if got := archiveFallbackURLs(in); !reflect.DeepEqual(got, want) {
			t.Errorf("archiveFallbackURLs(%q) = %v, want %v", in, got, want)
		}
	}
}

func withCatalogEntry(t *testing.T, name string, spec DownloadSpec) {
	t.Helper()
	old, had := DownloadCatalog[name]
	DownloadCatalog[name] = spec
	t.Cleanup(func() {
		if had {
			DownloadCatalog[name] = old
		} else {
			delete(DownloadCatalog, name)
		}
	})
}

func TestDownloadCandidatesOrder(t *testing.T) {
	withCatalogEntry(t, "TestPkg", DownloadSpec{
		Version: "8.4.1", URL: "https://windows.php.net/downloads/releases/old.zip", FileName: "old.zip",
		URLResolver: func(func(string)) (string, string, string, string, error) {
			return "https://example.test/new.zip", "new.zip", "new/", "8.4.9", nil
		},
	})
	got, err := DownloadCandidates("TestPkg", "", func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	var urls []string
	for _, c := range got {
		urls = append(urls, c.URL)
	}
	want := []string{
		"https://example.test/new.zip",
		"https://windows.php.net/downloads/releases/old.zip",
		"https://downloads.php.net/~windows/releases/archives/old.zip",
		"https://windows.php.net/downloads/releases/archives/old.zip",
	}
	if !reflect.DeepEqual(urls, want) {
		t.Fatalf("candidates = %v, want %v", urls, want)
	}
	if got[0].StripTop != "new/" || got[1].FileName != "old.zip" {
		t.Errorf("candidate metadata mixed up: %+v", got[:2])
	}
}

func TestDownloadCandidatesResolverFailureKeepsPinned(t *testing.T) {
	withCatalogEntry(t, "TestPkg", DownloadSpec{
		Version: "1.0", URL: "https://example.test/pinned.zip", FileName: "pinned.zip",
		URLResolver: func(func(string)) (string, string, string, string, error) {
			return "", "", "", "", errors.New("offline")
		},
	})
	got, err := DownloadCandidates("TestPkg", "", func(string) {})
	if err != nil || len(got) != 1 || got[0].URL != "https://example.test/pinned.zip" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestFetchFirstCandidateSkips404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/gone.zip":
			http.NotFound(w, r)
		case "/page.zip":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html>removed</html>"))
		default:
			w.Header().Set("Content-Type", "application/zip")
			w.Write([]byte("PK\x03\x04payload"))
		}
	}))
	defer srv.Close()

	cache := newDownloadCache(t.TempDir(), nil)
	path, used, err := FetchFirstCandidate(cache, "TestPkg", []downloadCandidate{
		{Source: "latest", URL: srv.URL + "/gone.zip", FileName: "gone.zip"},
		{Source: "pinned", URL: srv.URL + "/page.zip", FileName: "page.zip"},
		{Source: "pinned (archive)", URL: srv.URL + "/ok.zip", FileName: "ok.zip"},
	}, func(string) {}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if used.FileName != "ok.zip" || path != cache.Path("ok.zip") {
		t.Errorf("used %+v at %s", used, path)
	}
}

func TestEDBPostgresVersionRe(t *testing.T) {
	page := `<div>Binaries from installer<span class="x">Version <!-- -->16.15</span>`
	if m := EdbPostgres16Re.FindStringSubmatch(page); len(m) != 2 || m[1] != "16.15" {
		t.Fatalf("match = %v", m)
	}
}

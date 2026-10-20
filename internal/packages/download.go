//go:build windows

package packages

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"goampp/internal/pathenv"
)

var DownloadMu sync.Mutex

type ProgressFunc func(stage, name string, done, total int64)

func NopProgress(stage, name string, done, total int64) {}

func DownloadAndInstall(name, baseDir string, log func(string), progress ProgressFunc) error {
	return DownloadAndInstallVersion(name, "", baseDir, log, progress)
}

func DownloadAndInstallVersion(name, version, baseDir string, log func(string), progress ProgressFunc) error {
	spec, ok := DownloadCatalog[name]
	if !ok {
		return fmt.Errorf("no download info registered for %q", name)
	}

	notes := spec.Notes
	// The version actually installed is only known once a candidate is
	// resolved (logged as "source: ..."); name a branch only when one was picked.
	versionLabel := ""
	if len(spec.Variants) > 0 && version != "" {
		for _, v := range spec.Variants {
			if v.Version == version && v.Notes != "" {
				notes = v.Notes
			}
		}
		versionLabel = version
	}

	DownloadMu.Lock()
	defer DownloadMu.Unlock()

	canonicalDir := filepath.Join(baseDir, filepath.FromSlash(spec.InstallDir))
	installDir := canonicalDir
	if len(spec.Variants) > 0 && version != "" {
		installDir = canonicalDir + "-" + version
	}

	if spec.CheckFile != "" {
		if _, err := os.Stat(filepath.Join(installDir, filepath.FromSlash(spec.CheckFile))); err == nil {
			log(fmt.Sprintf("[%s] already installed at %s", name, installDir))

			if installDir != canonicalDir {
				if err := pointJunction(canonicalDir, installDir, log); err != nil {
					return err
				}
			}

			if spec.PostInstall != nil {
				if err := spec.PostInstall(canonicalDir, log); err != nil {
					return fmt.Errorf("post-install: %w", err)
				}
			}
			return nil
		}
	}

	if progress == nil {
		progress = NopProgress
	}
	progress("starting", name, 0, 0)

	if versionLabel != "" {
		log(fmt.Sprintf("[%s] %s — starting install", name, versionLabel))
	} else {
		log(fmt.Sprintf("[%s] starting install", name))
	}
	if notes != "" {
		log("  note: " + notes)
	}

	candidates, err := DownloadCandidates(name, version, log)
	if err != nil {
		progress("idle", name, 0, 0)
		return err
	}
	dlPath, used, err := FetchFirstCandidate(CacheFor(baseDir, log), name, candidates, log, func(done, total int64) {
		progress("downloading", name, done, total)
	})
	if err != nil {
		progress("idle", name, 0, 0)
		return fmt.Errorf("download: %w", err)
	}
	fileName, stripTop := used.FileName, used.StripTop
	if version == "" {
		recordInstalledCandidate(name, used)
	}

	if err := EnsureInstallDir(installDir); err != nil {
		progress("idle", name, 0, 0)
		return fmt.Errorf("create install dir: %w", err)
	}

	switch spec.Kind {
	case Zip:
		log(fmt.Sprintf("  extracting into %s", installDir))
		progress("extracting", name, 0, 0)
		if err := ExtractZip(dlPath, installDir, stripTop, func(done, total int64) {
			progress("extracting", name, done, total)
		}); err != nil {
			progress("idle", name, 0, 0)
			return fmt.Errorf("extract: %w", err)
		}
	case File:
		target := spec.TargetFile
		if target == "" {
			target = fileName
		}
		log(fmt.Sprintf("  copying to %s/%s", installDir, target))
		if err := CopyFile(dlPath, filepath.Join(installDir, target)); err != nil {
			progress("idle", name, 0, 0)
			return fmt.Errorf("copy: %w", err)
		}
	case Exe:
		log(fmt.Sprintf("  running silent installer → %s (this may take 30–60s)", installDir))
		progress("post-install", name, 0, 0)
		absInstallDir, _ := filepath.Abs(installDir)
		cmd := exec.Command(dlPath, "/S", "/D="+absInstallDir)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		if out, err := cmd.CombinedOutput(); err != nil {
			log("  installer output: " + strings.TrimSpace(string(out)))
			progress("idle", name, 0, 0)
			return fmt.Errorf("silent installer: %w", err)
		}
		log("  installer finished")
	default:
		progress("idle", name, 0, 0)
		return fmt.Errorf("unknown kind: %q", spec.Kind)
	}

	if installDir != canonicalDir {
		if err := pointJunction(canonicalDir, installDir, log); err != nil {
			return err
		}
	}

	if spec.PostInstall != nil {
		progress("post-install", name, 0, 0)
		if err := spec.PostInstall(canonicalDir, log); err != nil {
			progress("idle", name, 0, 0)
			return fmt.Errorf("post-install: %w", err)
		}
	}

	// Never report success for an install that left the check file missing:
	// the service would show as not installed and the next click would
	// silently download it again.
	if spec.CheckFile != "" {
		if _, err := os.Stat(filepath.Join(canonicalDir, filepath.FromSlash(spec.CheckFile))); err != nil {
			progress("idle", name, 0, 0)
			return fmt.Errorf("install finished but %s is missing in %s — the package layout may have changed", spec.CheckFile, canonicalDir)
		}
	}

	log(fmt.Sprintf("[%s] install complete", name))

	if n, err := pathenv.AddGoamppToUserPath(); err != nil {
		log(fmt.Sprintf("[%s] PATH update skipped: %v", name, err))
	} else if n > 0 {
		log(fmt.Sprintf("[%s] added %d bin dir(s) to user PATH — open a new terminal to use them", name, n))
	}

	progress("done", name, 0, 0)
	time.Sleep(500 * time.Millisecond)
	progress("idle", name, 0, 0)
	return nil
}

func pointJunction(canonical, target string, log func(string)) error {

	if fi, err := os.Lstat(canonical); err == nil {
		isReparse := fi.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0
		if isReparse {
			if err := os.Remove(canonical); err != nil {
				return fmt.Errorf("remove old junction: %w", err)
			}
			log("  removed legacy junction at " + canonical)
		} else if fi.IsDir() {

		}
	}
	if err := os.MkdirAll(canonical, 0o755); err != nil {
		return fmt.Errorf("create canonical dir: %w", err)
	}

	cmd := exec.Command("robocopy", target, canonical,
		"/MIR", "/NJH", "/NJS", "/NFL", "/NDL", "/NP", "/R:1", "/W:1")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	if err != nil {

		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if code >= 0 && code < 8 {
				err = nil
			}
		}
	}
	if err != nil {
		return fmt.Errorf("robocopy mirror %s → %s: %v: %s",
			target, canonical, err, strings.TrimSpace(string(out)))
	}
	log(fmt.Sprintf("  active version → %s", filepath.Base(target)))
	return nil
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func EnsureInstallDir(path string) error {
	if fi, err := os.Lstat(path); err == nil {
		if fi.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {

			real, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("resolve junction %s: %w", path, err)
			}
			return os.MkdirAll(real, 0o755)
		}
		if fi.IsDir() {
			return nil
		}
		return fmt.Errorf("%s exists but is not a directory", path)
	}
	return os.MkdirAll(path, 0o755)
}

func SetActiveVariant(name, version, baseDir string, log func(string), progress ProgressFunc) error {
	spec, ok := DownloadCatalog[name]
	if !ok {
		return fmt.Errorf("no catalogue entry for %q", name)
	}
	if len(spec.Variants) == 0 {
		return fmt.Errorf("[%s] is not multi-version", name)
	}

	return DownloadAndInstallVersion(name, version, baseDir, log, progress)
}

func HttpDownload(url, dest string, log func(string), onProgress func(done, total int64)) error {
	log("  GET " + url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "GoAMPP/0.3 (+https://github.com/goampp)")

	client := &http.Client{

		Timeout: 10 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	// Some mirrors (Apache Lounge) answer a dead build with 200 + a small HTML
	// error page. left unguarded that lands in downloads/ as a bogus .zip and
	// every later retry fails at extract off the cached copy.
	if ct := resp.Header.Get("Content-Type"); strings.Contains(strings.ToLower(ct), "html") {
		return fmt.Errorf("expected %s but got Content-Type %q — the server returned a page, not the file (URL likely stale)", filepath.Ext(dest), ct)
	}

	total := resp.ContentLength
	if total > 0 {
		log(fmt.Sprintf("  size: %.1f MB", float64(total)/(1024*1024)))
	}

	partPath := dest + ".part"
	f, err := os.Create(partPath)
	if err != nil {
		return err
	}

	closed := false
	defer func() {
		if !closed {
			f.Close()
			os.Remove(partPath)
		}
	}()

	buf := make([]byte, 128*1024)
	var read int64
	lastLog := time.Now()
	lastProgress := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			read += int64(n)

			if time.Since(lastLog) > time.Second {
				if total > 0 {
					pct := float64(read) * 100 / float64(total)
					log(fmt.Sprintf("  %5.1f%%  %6.1f / %.1f MB",
						pct, float64(read)/(1024*1024), float64(total)/(1024*1024)))
				} else {
					log(fmt.Sprintf("  downloaded %.1f MB", float64(read)/(1024*1024)))
				}
				lastLog = time.Now()
			}

			if onProgress != nil && time.Since(lastProgress) > 33*time.Millisecond {
				onProgress(read, total)
				lastProgress = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}

	if onProgress != nil {
		onProgress(read, total)
	}

	if err := f.Close(); err != nil {
		return err
	}
	closed = true

	_ = os.Remove(dest)
	return os.Rename(partPath, dest)
}

func ExtractZip(zipPath, destDir, stripTop string, onProgress func(done, total int64)) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return err
	}

	stripTop = effectiveStripTop(r.File, stripTop)

	total := int64(len(r.File))
	var done int64
	for _, f := range r.File {
		done++
		if onProgress != nil {
			onProgress(done, total)
		}
		name := f.Name
		if stripTop != "" {
			if !strings.HasPrefix(name, stripTop) {
				continue
			}
			name = strings.TrimPrefix(name, stripTop)
		}
		if name == "" {
			continue
		}

		target := filepath.Join(absDest, filepath.FromSlash(name))

		absTarget, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(absTarget, absDest+string(os.PathSeparator)) && absTarget != absDest {
			return fmt.Errorf("zip entry escapes destination: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(absTarget, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(absTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		out.Close()
		rc.Close()
	}
	return nil
}

// effectiveStripTop checks the expected top-level folder against the archive.
// Resolvers guess it from the file name, and upstream layouts change between
// releases (Crystal 1.22 zips its files at the root), so when no entry starts
// with the guess, the archive's own single top folder is stripped instead, or
// nothing when the files sit at the root.
func effectiveStripTop(files []*zip.File, stripTop string) string {
	if stripTop == "" {
		return ""
	}
	top := ""
	single := true
	for _, f := range files {
		if strings.HasPrefix(f.Name, stripTop) {
			return stripTop
		}
		i := strings.Index(f.Name, "/")
		if i < 0 {
			single = false
			continue
		}
		if top == "" {
			top = f.Name[:i+1]
		} else if top != f.Name[:i+1] {
			single = false
		}
	}
	if single && top != "" {
		return top
	}
	return ""
}

func CopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func IsInstalled(name, baseDir string) bool {
	spec, ok := DownloadCatalog[name]
	if !ok {
		return true
	}
	if spec.CheckFile == "" {
		return false
	}
	installDir := filepath.Join(baseDir, filepath.FromSlash(spec.InstallDir))
	_, err := os.Stat(filepath.Join(installDir, filepath.FromSlash(spec.CheckFile)))
	return err == nil
}

//go:build linux

package linuxapp

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type projectRunSpec struct {
	Command  string
	Args     []string
	URL      string
	Port     int
	PortMode string
}

type managedProject struct {
	cmd  *exec.Cmd
	done chan struct{}
}

func (p *managedProject) finished() bool {
	if p == nil {
		return true
	}
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

var (
	runningProjectsMu sync.Mutex
	runningProjects   = make(map[string]*managedProject)
)

func availableLocalPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected local listener address type %T", listener.Addr())
	}
	return addr.Port, nil
}

func projectRunSpecAtPort(spec projectRunSpec, port int) projectRunSpec {
	spec.Port = port
	spec.URL = fmt.Sprintf("http://127.0.0.1:%d", port)
	if spec.PortMode == "php" {
		spec.Args = []string{"-S", fmt.Sprintf("127.0.0.1:%d", port), "-t", "public", "router.php"}
	}
	return spec
}

func environmentWith(key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	return append(out, prefix+value)
}

func projectRunSpecForDir(dir string) (projectRunSpec, error) {
	has := func(name string) bool {
		info, err := os.Stat(filepath.Join(dir, name))
		return err == nil && info.Mode().IsRegular()
	}
	switch {
	case has("package.json") && has("server.js"):
		return projectRunSpec{Command: "npm", Args: []string{"start"}, URL: "http://127.0.0.1:3000", Port: 3000, PortMode: "env"}, nil
	case has("go.mod") && has("main.go"):
		return projectRunSpec{Command: "go", Args: []string{"run", "."}, URL: "http://127.0.0.1:8080", Port: 8080, PortMode: "env"}, nil
	case has("serve.py") && has("public/index.html"):
		command := "python3"
		if _, err := exec.LookPath(command); err != nil {
			command = "python"
		}
		return projectRunSpec{Command: command, Args: []string{"serve.py"}, URL: "http://127.0.0.1:8000", Port: 8000, PortMode: "env"}, nil
	case has("server.py"):
		command := "python3"
		if _, err := exec.LookPath(command); err != nil {
			command = "python"
		}
		return projectRunSpec{Command: command, Args: []string{"server.py"}, URL: "http://127.0.0.1:8002", Port: 8002, PortMode: "env"}, nil
	case has("public/index.php"):
		return projectRunSpec{Command: "php", Args: []string{"-S", "127.0.0.1:8001", "-t", "public", "router.php"}, URL: "http://127.0.0.1:8001", Port: 8001, PortMode: "php"}, nil
	default:
		return projectRunSpec{}, fmt.Errorf("no supported GoAMPP starter was detected in this folder")
	}
}

func startProject(root, name string) (string, error) {
	resolvedPath, err := resolveProjectDir(root, name)
	if err != nil {
		return "", err
	}
	spec, err := projectRunSpecForDir(resolvedPath)
	if err != nil {
		return "", err
	}
	port, err := availableLocalPort()
	if err != nil {
		return "", fmt.Errorf("choose a local preview port: %w", err)
	}
	spec = projectRunSpecAtPort(spec, port)
	binary, err := exec.LookPath(spec.Command)
	if err != nil {
		return "", fmt.Errorf("%s is not installed or not available on PATH", spec.Command)
	}
	runningProjectsMu.Lock()
	if managed := runningProjects[resolvedPath]; managed != nil && !managed.finished() {
		runningProjectsMu.Unlock()
		return "", fmt.Errorf("this project is already running")
	}
	runningProjectsMu.Unlock()

	logPath := filepath.Join(resolvedPath, ".goampp-run.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return "", fmt.Errorf("open project log: %w", err)
	}

	cmd := exec.Command(binary, spec.Args...)
	cmd.Dir = resolvedPath
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if spec.PortMode == "env" {
		cmd.Env = environmentWith("GOAMPP_PORT", strconv.Itoa(spec.Port))
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	runningProjectsMu.Lock()
	if managed := runningProjects[resolvedPath]; managed != nil && !managed.finished() {
		runningProjectsMu.Unlock()
		_ = logFile.Close()
		return "", fmt.Errorf("this project is already running")
	}
	if err := cmd.Start(); err != nil {
		runningProjectsMu.Unlock()
		_ = logFile.Close()
		return "", fmt.Errorf("start project: %w", err)
	}
	managed := &managedProject{cmd: cmd, done: make(chan struct{})}
	runningProjects[resolvedPath] = managed
	runningProjectsMu.Unlock()
	_ = logFile.Close()

	superviseProject(resolvedPath, logPath, cmd, managed)
	return spec.URL, nil
}

func resolveProjectDir(root, name string) (string, error) {
	if !projectNamePattern.MatchString(name) {
		return "", fmt.Errorf("invalid project directory name")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rootAbs, err = filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", fmt.Errorf("resolve projects directory: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(filepath.Join(rootAbs, name))
	if err != nil {
		return "", fmt.Errorf("resolve project folder: %w", err)
	}
	relative, err := filepath.Rel(rootAbs, resolvedPath)
	if err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || filepath.IsAbs(relative) {
		return "", fmt.Errorf("project path is outside the projects directory")
	}
	info, err := os.Stat(resolvedPath)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("project folder does not exist")
	}
	return resolvedPath, nil
}

func isProjectRunning(root, name string) bool {
	resolvedPath, err := resolveProjectDir(root, name)
	if err != nil {
		return false
	}
	runningProjectsMu.Lock()
	managed := runningProjects[resolvedPath]
	runningProjectsMu.Unlock()
	return managed != nil && !managed.finished()
}

func stopProject(root, name string) error {
	resolvedPath, err := resolveProjectDir(root, name)
	if err != nil {
		return err
	}
	runningProjectsMu.Lock()
	managed := runningProjects[resolvedPath]
	runningProjectsMu.Unlock()
	if managed == nil || managed.finished() || managed.cmd == nil || managed.cmd.Process == nil {
		return fmt.Errorf("this project is not running")
	}

	pid := managed.cmd.Process.Pid
	if err := terminateProcessGroup(pid, 600*time.Millisecond); err != nil {
		_ = managed.cmd.Process.Kill()
		if !waitForManagedProjectExit(managed, time.Second) {
			return fmt.Errorf("stop project: %w; timed out waiting for its process to exit", err)
		}
		return fmt.Errorf("stop project: %w", err)
	}
	if !waitForManagedProjectExit(managed, time.Second) {
		return fmt.Errorf("timed out waiting for the project process group to exit")
	}
	return nil
}

func superviseProject(resolvedPath, logPath string, cmd *exec.Cmd, managed *managedProject) {
	go func() {
		waitErr := cmd.Wait()
		groupErr := terminateProcessGroup(cmd.Process.Pid, 250*time.Millisecond)
		if file, openErr := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0o600); openErr == nil {
			if waitErr != nil {
				_, _ = fmt.Fprintf(file, "\n[GoAMPP] process exited: %v\n", waitErr)
			} else {
				_, _ = fmt.Fprintln(file, "\n[GoAMPP] process exited")
			}
			if groupErr != nil {
				_, _ = fmt.Fprintf(file, "[GoAMPP] process group cleanup failed: %v\n", groupErr)
			}
			_ = file.Close()
		}
		close(managed.done)
		runningProjectsMu.Lock()
		if runningProjects[resolvedPath] == managed {
			delete(runningProjects, resolvedPath)
		}
		runningProjectsMu.Unlock()
	}()
}

func processGroupExists(pgid int) bool {
	if pgid <= 0 {
		return false
	}
	entries, err := os.ReadDir("/proc")
	if err == nil {
		uncertain := false
		for _, entry := range entries {
			if _, parseErr := strconv.Atoi(entry.Name()); parseErr != nil {
				continue
			}
			data, readErr := os.ReadFile(filepath.Join("/proc", entry.Name(), "stat"))
			if readErr != nil {
				if !errors.Is(readErr, os.ErrNotExist) {
					uncertain = true
				}
				continue
			}
			state, group, ok := parseLinuxProcStat(string(data))
			if ok && group == pgid && state != 'Z' && state != 'X' {
				return true
			}
		}
		if !uncertain {
			return false
		}
	}
	// Fall back when /proc is unavailable or does not expose every process.
	// SIGCONT is not sent; signal zero only checks process-group existence.
	checkErr := syscall.Kill(-pgid, 0)
	return checkErr == nil || !errors.Is(checkErr, syscall.ESRCH)
}

func parseLinuxProcStat(stat string) (state byte, processGroup int, ok bool) {
	closeParen := strings.LastIndexByte(stat, ')')
	if closeParen < 0 || closeParen+1 >= len(stat) {
		return 0, 0, false
	}
	fields := strings.Fields(stat[closeParen+1:])
	if len(fields) < 3 || len(fields[0]) != 1 {
		return 0, 0, false
	}
	group, err := strconv.Atoi(fields[2])
	if err != nil {
		return 0, 0, false
	}
	return fields[0][0], group, true
}

func terminateProcessGroup(pgid int, grace time.Duration) error {
	if pgid <= 0 {
		return fmt.Errorf("invalid process group ID %d", pgid)
	}
	// Do not signal a stale process-group ID after its leader has exited and
	// the entire group has disappeared; the numeric ID could be reused.
	if !processGroupExists(pgid) {
		return nil
	}
	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return fmt.Errorf("send SIGTERM to process group %d: %w", pgid, err)
	}

	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if !processGroupExists(pgid) {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !processGroupExists(pgid) {
		return nil
	}
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("send SIGKILL to process group %d: %w", pgid, err)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if !processGroupExists(pgid) {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	if processGroupExists(pgid) {
		return fmt.Errorf("process group %d did not exit after SIGKILL", pgid)
	}
	return nil
}

func waitForManagedProjectExit(managed *managedProject, timeout time.Duration) bool {
	if managed == nil || managed.done == nil {
		return false
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-managed.done:
		return true
	case <-timer.C:
		return false
	}
}

func openPreviewWhenReady(projectDir, projectName, previewURL string, onDone func(error)) {
	go func() {
		err := waitForProjectEndpoint(projectDir, projectName, previewURL, 15*time.Second)
		if err == nil {
			err = openExternal(previewURL)
		}
		if err != nil {
			appendProjectLog(projectDir, projectName, "[GoAMPP] preview: "+err.Error())
		}
		if onDone != nil {
			onDone(err)
		}
	}()
}

func waitForProjectEndpoint(root, name, rawURL string, timeout time.Duration) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid preview URL")
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if !isProjectRunning(root, name) {
			return fmt.Errorf("project process exited before its preview became ready")
		}
		conn, err := net.DialTimeout("tcp", parsed.Host, 150*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			if !isProjectRunning(root, name) {
				return fmt.Errorf("project process exited while its preview was starting")
			}
			return nil
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for %s", parsed.Host)
	}
	return fmt.Errorf("waiting for %s: %w", parsed.Host, lastErr)
}

func appendProjectLog(projectDir, projectName, line string) {
	if !projectNamePattern.MatchString(projectName) {
		return
	}
	path := filepath.Join(projectDir, projectName, ".goampp-run.log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintln(file, line)
	_ = file.Close()
}

func stopAllProjects() {
	runningProjectsMu.Lock()
	projects := make([]*managedProject, 0, len(runningProjects))
	for _, managed := range runningProjects {
		if managed != nil && !managed.finished() && managed.cmd != nil && managed.cmd.Process != nil {
			projects = append(projects, managed)
		}
	}
	runningProjectsMu.Unlock()

	for _, managed := range projects {
		if !managed.finished() && processGroupExists(managed.cmd.Process.Pid) {
			_ = syscall.Kill(-managed.cmd.Process.Pid, syscall.SIGTERM)
		}
	}
	if len(projects) > 0 {
		time.Sleep(250 * time.Millisecond)
	}
	for _, managed := range projects {
		if managed.finished() || managed.cmd == nil || managed.cmd.Process == nil {
			continue
		}
		pid := managed.cmd.Process.Pid
		if processGroupExists(pid) {
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				_ = managed.cmd.Process.Kill()
			}
		}
	}
	deadline := time.Now().Add(time.Second)
	for _, managed := range projects {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		_ = waitForManagedProjectExit(managed, remaining)
	}
}

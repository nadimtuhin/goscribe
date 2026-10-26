package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Runner abstracts process execution and system process operations.
type Runner interface {
	Which(bin string) (string, error)
	Run(ctx context.Context, cmd []string) (stdout string, stderr string, exitCode int, err error)
	SpawnDetached(bin string, args []string, logFile string) (pid int, err error)
	Kill(pid int, sig syscall.Signal) error
	IsAlive(pid int) bool
	IsRunning(pid int, executableName string) bool
	Sleep(d time.Duration)
}

// RealRunner implements Runner with real OS process operations.
type RealRunner struct{}

var DefaultRunner Runner = &RealRunner{}

func (r *RealRunner) Which(bin string) (string, error) {
	return exec.LookPath(bin)
}

func (r *RealRunner) Run(ctx context.Context, cmd []string) (stdout string, stderr string, exitCode int, err error) {
	if len(cmd) == 0 {
		return "", "", -1, errors.New("empty command")
	}
	c := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stdout = &stdoutBuf
	c.Stderr = &stderrBuf

	err = c.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()
	if err == nil {
		return stdout, stderr, 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			exitCode = 1
		} else if exitCode < 0 {
			exitCode = 1
		}
		return stdout, stderr, exitCode, err
	}

	return stdout, stderr, -1, err
}

func (r *RealRunner) SpawnDetached(bin string, args []string, logFile string) (pid int, err error) {
	if dir := filepath.Dir(logFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return 0, err
		}
	}
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	cmd := exec.Command(bin, args...)
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	return cmd.Process.Pid, nil
}

func (r *RealRunner) Kill(pid int, sig syscall.Signal) error {
	if pid <= 0 {
		return syscall.ESRCH
	}
	return syscall.Kill(pid, sig)
}

func (r *RealRunner) IsAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func (r *RealRunner) IsRunning(pid int, executableName string) bool {
	if !r.IsAlive(pid) {
		return false
	}
	out, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return false
	}
	comm := strings.TrimSpace(string(out))
	if comm == "" {
		return false
	}
	return strings.HasSuffix(comm, executableName) || filepath.Base(comm) == executableName
}

func (r *RealRunner) Sleep(d time.Duration) {
	time.Sleep(d)
}

package runner

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// SpawnCall records parameters for SpawnDetached invocations.
type SpawnCall struct {
	Bin     string
	Args    []string
	LogFile string
}

// KillCall records parameters for Kill invocations.
type KillCall struct {
	Pid int
	Sig syscall.Signal
}

// IsRunningCall records parameters for IsRunning invocations.
type IsRunningCall struct {
	Pid            int
	ExecutableName string
}

// FakeRunner implements Runner for unit testing other packages.
type FakeRunner struct {
	mu sync.Mutex

	WhichFn         func(bin string) (string, error)
	RunFn           func(ctx context.Context, cmd []string) (stdout string, stderr string, exitCode int, err error)
	SpawnDetachedFn func(bin string, args []string, logFile string) (pid int, err error)
	KillFn          func(pid int, sig syscall.Signal) error
	IsAliveFn       func(pid int) bool
	IsRunningFn     func(pid int, executableName string) bool
	SleepFn         func(d time.Duration)

	WhichCalls         []string
	RunCalls           [][]string
	SpawnDetachedCalls []SpawnCall
	KillCalls          []KillCall
	IsAliveCalls       []int
	IsRunningCalls     []IsRunningCall
	SleepCalls         []time.Duration

	Alive   map[int]string
	NextPID int
}

var _ Runner = (*FakeRunner)(nil)
var _ Runner = (*RealRunner)(nil)

func NewFakeRunner() *FakeRunner {
	return &FakeRunner{
		Alive:   make(map[int]string),
		NextPID: 1000,
	}
}

func (f *FakeRunner) ensureInitialized() {
	if f.Alive == nil {
		f.Alive = make(map[int]string)
	}
	if f.NextPID == 0 {
		f.NextPID = 1000
	}
}

func (f *FakeRunner) Which(bin string) (string, error) {
	f.mu.Lock()
	f.WhichCalls = append(f.WhichCalls, bin)
	fn := f.WhichFn
	f.mu.Unlock()

	if fn != nil {
		return fn(bin)
	}
	return "/fake/bin/" + bin, nil
}

func (f *FakeRunner) Run(ctx context.Context, cmd []string) (stdout string, stderr string, exitCode int, err error) {
	f.mu.Lock()
	cmdCopy := make([]string, len(cmd))
	copy(cmdCopy, cmd)
	f.RunCalls = append(f.RunCalls, cmdCopy)
	fn := f.RunFn
	f.mu.Unlock()

	if fn != nil {
		return fn(ctx, cmd)
	}
	return "", "", 0, nil
}

func (f *FakeRunner) SpawnDetached(bin string, args []string, logFile string) (pid int, err error) {
	f.mu.Lock()
	f.ensureInitialized()
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)
	f.SpawnDetachedCalls = append(f.SpawnDetachedCalls, SpawnCall{
		Bin:     bin,
		Args:    argsCopy,
		LogFile: logFile,
	})
	fn := f.SpawnDetachedFn
	if fn != nil {
		f.mu.Unlock()
		return fn(bin, args, logFile)
	}

	pid = f.NextPID
	f.NextPID++
	f.Alive[pid] = filepath.Base(bin)
	f.mu.Unlock()
	return pid, nil
}

func (f *FakeRunner) Kill(pid int, sig syscall.Signal) error {
	f.mu.Lock()
	f.ensureInitialized()
	f.KillCalls = append(f.KillCalls, KillCall{Pid: pid, Sig: sig})
	fn := f.KillFn
	if fn != nil {
		f.mu.Unlock()
		return fn(pid, sig)
	}

	delete(f.Alive, pid)
	f.mu.Unlock()
	return nil
}

func (f *FakeRunner) IsAlive(pid int) bool {
	f.mu.Lock()
	f.ensureInitialized()
	f.IsAliveCalls = append(f.IsAliveCalls, pid)
	fn := f.IsAliveFn
	if fn != nil {
		f.mu.Unlock()
		return fn(pid)
	}

	_, ok := f.Alive[pid]
	f.mu.Unlock()
	return ok
}

func (f *FakeRunner) IsRunning(pid int, executableName string) bool {
	f.mu.Lock()
	f.ensureInitialized()
	f.IsRunningCalls = append(f.IsRunningCalls, IsRunningCall{Pid: pid, ExecutableName: executableName})
	fn := f.IsRunningFn
	if fn != nil {
		f.mu.Unlock()
		return fn(pid, executableName)
	}

	comm, ok := f.Alive[pid]
	f.mu.Unlock()
	if !ok {
		return false
	}
	return comm == executableName || strings.HasSuffix(comm, executableName) || filepath.Base(comm) == executableName
}

func (f *FakeRunner) Sleep(d time.Duration) {
	f.mu.Lock()
	f.SleepCalls = append(f.SleepCalls, d)
	fn := f.SleepFn
	f.mu.Unlock()

	if fn != nil {
		fn(d)
	}
}

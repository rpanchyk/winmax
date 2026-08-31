//go:build windows

package session

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	workerArg  = "worker"
	minBackoff = 2 * time.Second
	maxBackoff = 60 * time.Second
)

type slot struct {
	handle  windows.Handle
	nextTry time.Time
	backoff time.Duration
}

type Supervisor struct {
	log *log.Logger

	mu      sync.Mutex
	workers map[uint32]slot
}

func New(logger *log.Logger) *Supervisor {
	if logger == nil {
		logger = log.Default()
	}
	return &Supervisor{
		log:     logger,
		workers: make(map[uint32]slot),
	}
}

func (s *Supervisor) Sync() {
	ids, err := activeSessions()
	if err != nil {
		s.log.Printf("enumerate sessions: %v", err)
		return
	}

	want := make(map[uint32]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, sl := range s.workers {
		if _, ok := want[id]; ok {
			continue
		}
		stopHandle(sl.handle)
		delete(s.workers, id)
	}

	now := time.Now()
	for id := range want {
		sl := s.workers[id]
		if sl.handle != 0 && alive(sl.handle) {
			continue
		}
		if sl.handle != 0 {
			code := exitCode(sl.handle)
			stopHandle(sl.handle)
			sl.handle = 0
			sl.backoff = nextBackoff(sl.backoff)
			sl.nextTry = now.Add(sl.backoff)
			s.workers[id] = sl
			s.log.Printf("worker in session %d exited (code %d); retry in %s", id, code, sl.backoff)
			continue
		}
		if !sl.nextTry.IsZero() && now.Before(sl.nextTry) {
			continue
		}

		h, err := startWorker(id)
		if err != nil {
			sl.backoff = nextBackoff(sl.backoff)
			sl.nextTry = now.Add(sl.backoff)
			s.workers[id] = sl
			s.log.Printf("start worker in session %d: %v (retry in %s)", id, err, sl.backoff)
			continue
		}
		sl.handle = h
		s.workers[id] = sl
		s.log.Printf("started worker in session %d", id)
	}
}

func (s *Supervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sl := range s.workers {
		stopHandle(sl.handle)
		delete(s.workers, id)
	}
}

func HasInteractiveSession() bool {
	ids, err := activeSessions()
	return err == nil && len(ids) > 0
}

func (s *Supervisor) AliveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, sl := range s.workers {
		if sl.handle != 0 && alive(sl.handle) {
			n++
		}
	}
	return n
}

func startWorker(sessionID uint32) (windows.Handle, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return 0, err
	}

	var token windows.Token
	if err := windows.WTSQueryUserToken(sessionID, &token); err != nil {
		return 0, fmt.Errorf("query user token: %w", err)
	}
	defer token.Close()

	var env *uint16
	if err := windows.CreateEnvironmentBlock(&env, token, false); err == nil {
		defer windows.DestroyEnvironmentBlock(env)
	} else {
		env = nil
	}

	app, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return 0, err
	}
	cmd, err := windows.UTF16PtrFromString(`"` + exe + `" ` + workerArg)
	if err != nil {
		return 0, err
	}
	dir, err := windows.UTF16PtrFromString(filepath.Dir(exe))
	if err != nil {
		return 0, err
	}
	desktop, err := windows.UTF16PtrFromString(`winsta0\default`)
	if err != nil {
		return 0, err
	}

	si := windows.StartupInfo{
		Flags:      windows.STARTF_USESHOWWINDOW,
		ShowWindow: uint16(windows.SW_HIDE),
		Desktop:    desktop,
	}
	si.Cb = uint32(unsafe.Sizeof(si))

	var pi windows.ProcessInformation
	err = windows.CreateProcessAsUser(
		token,
		app,
		cmd,
		nil,
		nil,
		false,
		windows.CREATE_UNICODE_ENVIRONMENT|windows.DETACHED_PROCESS|windows.CREATE_NEW_PROCESS_GROUP|windows.CREATE_NO_WINDOW,
		env,
		dir,
		&si,
		&pi,
	)
	if err != nil {
		return 0, fmt.Errorf("create process: %w", err)
	}
	windows.CloseHandle(pi.Thread)
	return pi.Process, nil
}

func activeSessions() ([]uint32, error) {
	var info *windows.WTS_SESSION_INFO
	var count uint32
	if err := windows.WTSEnumerateSessions(0, 0, 1, &info, &count); err != nil {
		id := windows.WTSGetActiveConsoleSessionId()
		if id == 0 || id == 0xFFFFFFFF {
			return nil, nil
		}
		return []uint32{id}, nil
	}
	defer windows.WTSFreeMemory(uintptr(unsafe.Pointer(info)))

	list := unsafe.Slice(info, count)
	ids := make([]uint32, 0, len(list))
	for _, session := range list {
		if session.SessionID != 0 && sessionWanted(session.State) {
			ids = append(ids, session.SessionID)
		}
	}
	return ids, nil
}

func sessionWanted(state uint32) bool {
	switch state {
	case windows.WTSActive, windows.WTSConnected, windows.WTSDisconnected:
		return true
	default:
		return false
	}
}

func nextBackoff(prev time.Duration) time.Duration {
	if prev < minBackoff {
		return minBackoff
	}
	next := prev * 2
	if next > maxBackoff {
		return maxBackoff
	}
	return next
}

func alive(h windows.Handle) bool {
	if h == 0 {
		return false
	}
	event, err := windows.WaitForSingleObject(h, 0)
	return err == nil && event == uint32(windows.WAIT_TIMEOUT)
}

func exitCode(h windows.Handle) uint32 {
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return 0
	}
	return code
}

func stopHandle(h windows.Handle) {
	if h == 0 {
		return
	}
	_ = windows.TerminateProcess(h, 1)
	_, _ = windows.WaitForSingleObject(h, 5000)
	_ = windows.CloseHandle(h)
}

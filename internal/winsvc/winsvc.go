//go:build windows

package winsvc

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"

	"winmax/internal/autostart"
	"winmax/internal/ipc"
	"winmax/internal/session"
	"winmax/internal/singleinstance"
)

const (
	Name        = "WinMax"
	DisplayName = "Window Maximizer"
	Description = "Maximizes new windows by rules in config.yml"
)

type handler struct{}

type eventWriter struct {
	elog *eventlog.Log
}

func (w eventWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\r\n")
	if msg != "" && w.elog != nil {
		_ = w.elog.Info(1, msg)
	}
	return len(p), nil
}

func Run() error {
	return svc.Run(Name, &handler{})
}

func (h *handler) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptSessionChange | svc.AcceptParamChange

	changes <- svc.Status{State: svc.StartPending}

	logger, cleanup := openServiceLogger()
	defer cleanup()
	globalReload, err := ipc.CreateGlobal()
	if err != nil {
		logger.Printf("global reload event: %v", err)
	} else {
		defer windows.CloseHandle(globalReload)
	}

	sup := session.New(logger)
	sup.Sync()
	logger.Printf("service started")

	changes <- svc.Status{State: svc.Running, Accepts: accepted}
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

loop:
	for {
		select {
		case <-tick.C:
			sup.Sync()
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.ParamChange:
				if err := ipc.PulseGlobal(); err != nil {
					logger.Printf("reload signal: %v", err)
				}
				sup.Sync()
			case svc.SessionChange:
				sup.Sync()
			case svc.Stop, svc.Shutdown:
				break loop
			default:
				changes <- c.CurrentStatus
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	sup.Stop()
	logger.Printf("service stopped")
	return false, 0
}

func Install(exePath string) error {
	if err := requireElevated(); err != nil {
		return err
	}
	if err := requireConfigBeside(exePath); err != nil {
		return err
	}
	_ = autostart.Uninstall()

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect service manager: %w", err)
	}
	defer m.Disconnect()

	cfg := mgr.Config{
		StartType:   mgr.StartAutomatic,
		DisplayName: DisplayName,
		Description: Description,
	}

	s, err := m.OpenService(Name)
	if err != nil {
		s, err = m.CreateService(Name, exePath, cfg)
		if err != nil {
			return fmt.Errorf("create service: %w", err)
		}
	} else {
		cur, queryErr := s.Config()
		if queryErr != nil {
			s.Close()
			return queryErr
		}
		cur.BinaryPathName = `"` + exePath + `"`
		cur.StartType = mgr.StartAutomatic
		cur.DisplayName = DisplayName
		cur.Description = Description
		if err := s.UpdateConfig(cur); err != nil {
			s.Close()
			return fmt.Errorf("update service: %w", err)
		}
	}
	defer s.Close()

	if err := ensureEventLog(); err != nil {
		return err
	}

	if err := restart(s); err != nil {
		return err
	}
	if err := singleinstance.WaitRunning(15 * time.Second); err != nil {
		if session.HasInteractiveSession() {
			return fmt.Errorf("service started but the desktop worker did not come up: %w (see Event Viewer source %s and winmax.log)", err, Name)
		}
	}
	return nil
}

func Uninstall() error {
	if err := requireElevated(); err != nil {
		return err
	}
	_ = autostart.Uninstall()

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(Name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", Name)
	}
	defer s.Close()

	_ = stopWait(s, 10*time.Second)
	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	_ = eventlog.Remove(Name)
	return nil
}

func Status() (string, error) {
	name, err := windows.UTF16PtrFromString(Name)
	if err != nil {
		return "", err
	}
	scm, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return "", fmt.Errorf("connect service manager: %w", err)
	}
	defer windows.CloseServiceHandle(scm)

	h, err := windows.OpenService(scm, name, windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)
	if err != nil {
		return "service: not installed", nil
	}
	defer windows.CloseServiceHandle(h)

	var t windows.SERVICE_STATUS_PROCESS
	var needed uint32
	err = windows.QueryServiceStatusEx(h, windows.SC_STATUS_PROCESS_INFO, (*byte)(unsafe.Pointer(&t)), uint32(unsafe.Sizeof(t)), &needed)
	if err != nil {
		return "", err
	}
	state := stateName(svc.State(t.CurrentState))

	n := uint32(1024)
	b := make([]byte, n)
	p := (*windows.QUERY_SERVICE_CONFIG)(unsafe.Pointer(&b[0]))
	if err := windows.QueryServiceConfig(h, p, n, &n); err != nil {
		return fmt.Sprintf("service: %s\nname: %s", state, Name), nil
	}
	return fmt.Sprintf("service: %s\nname: %s\nbinary: %s", state, Name, windows.UTF16PtrToString(p.BinaryPathName)), nil
}

func restart(s *mgr.Service) error {
	if err := stopWait(s, 10*time.Second); err != nil {
		return err
	}
	if err := s.Start(); err != nil {
		return fmt.Errorf("service installed, but start failed: %w", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st, err := s.Query()
		if err != nil {
			return err
		}
		if st.State == svc.Running {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for service to run")
}

func stopWait(s *mgr.Service, timeout time.Duration) error {
	status, err := s.Query()
	if err != nil {
		return err
	}
	if status.State == svc.Stopped {
		return nil
	}
	if _, err := s.Control(svc.Stop); err != nil && status.State != svc.StopPending {
		return fmt.Errorf("stop service: %w", err)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err = s.Query()
		if err != nil {
			return err
		}
		if status.State == svc.Stopped {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for service to stop")
}

func openServiceLogger() (*log.Logger, func()) {
	elog, err := eventlog.Open(Name)
	if err == nil {
		return log.New(eventWriter{elog}, "winmax: ", log.LstdFlags), func() { _ = elog.Close() }
	}
	return log.New(io.MultiWriter(os.Stderr), "winmax: ", log.LstdFlags), func() {}
}

func ensureEventLog() error {
	err := eventlog.InstallAsEventCreate(Name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err == nil {
		return nil
	}
	if el, openErr := eventlog.Open(Name); openErr == nil {
		_ = el.Close()
		return nil
	}
	return fmt.Errorf("event log source %s: %w", Name, err)
}

func requireConfigBeside(exePath string) error {
	path := filepath.Join(filepath.Dir(exePath), "config.yml")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("config.yml not found next to %s", exePath)
	}
	return nil
}

func requireElevated() error {
	if windows.GetCurrentProcessToken().IsElevated() {
		return nil
	}
	return fmt.Errorf("this command requires an elevated prompt (Run as administrator)")
}

func stateName(s svc.State) string {
	switch s {
	case svc.Stopped:
		return "stopped"
	case svc.StartPending:
		return "start pending"
	case svc.StopPending:
		return "stop pending"
	case svc.Running:
		return "running"
	default:
		return fmt.Sprintf("state %d", s)
	}
}

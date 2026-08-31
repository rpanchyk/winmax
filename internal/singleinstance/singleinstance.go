//go:build windows

package singleinstance

import (
	"fmt"
	"io"
	"time"

	"golang.org/x/sys/windows"
)

var mutexName = "Local\\WinMax_UserLogonDaemon"

type mutex struct {
	h windows.Handle
}

func (m *mutex) Close() error {
	if m.h == 0 {
		return nil
	}
	err := windows.CloseHandle(m.h)
	m.h = 0
	return err
}

func Running() bool {
	name, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return false
	}
	h, err := windows.OpenMutex(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return false
	}
	_ = windows.CloseHandle(h)
	return true
}

func WaitRunning(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if Running() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out after %s", timeout)
}

func Acquire() (io.Closer, error) {
	mu, err := windows.CreateMutex(nil, false, windows.StringToUTF16Ptr(mutexName))
	if err == windows.ERROR_ALREADY_EXISTS {
		if mu != 0 {
			_ = windows.CloseHandle(mu)
		}
		return nil, fmt.Errorf("winmax is already running")
	}
	if err != nil {
		return nil, fmt.Errorf("create mutex: %w", err)
	}
	return &mutex{h: mu}, nil
}

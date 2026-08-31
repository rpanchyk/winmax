//go:build windows

package ipc

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows"
)

var (
	localEvent  = "Local\\WinMax_Reload"
	globalEvent = "Global\\WinMax_Reload"
)

func RequestReload() error {
	localErr := pulse(localEvent)
	globalErr := pulse(globalEvent)
	if localErr == nil || globalErr == nil {
		return nil
	}
	return fmt.Errorf("daemon is not running")
}

func PulseGlobal() error {
	return pulse(globalEvent)
}

func CreateGlobal() (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(globalEvent)
	if err != nil {
		return 0, err
	}
	return windows.CreateEvent(nil, 0, 0, name)
}

func pulse(eventName string) error {
	name, err := windows.UTF16PtrFromString(eventName)
	if err != nil {
		return err
	}
	h, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, name)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	return windows.SetEvent(h)
}

func WatchReload(onReload func()) (stop func(), err error) {
	handles := make([]windows.Handle, 0, 2)

	local, err := createOrOpen(localEvent)
	if err != nil {
		return nil, fmt.Errorf("create reload event: %w", err)
	}
	handles = append(handles, local)

	if global, gerr := createOrOpen(globalEvent); gerr == nil {
		handles = append(handles, global)
	}

	done := make(chan struct{})
	go func() {
		defer func() {
			for _, h := range handles {
				_ = windows.CloseHandle(h)
			}
		}()
		for {
			event, waitErr := windows.WaitForMultipleObjects(handles, false, 200)
			select {
			case <-done:
				return
			default:
			}
			if waitErr != nil {
				continue
			}
			if event >= windows.WAIT_OBJECT_0 && event < windows.WAIT_OBJECT_0+uint32(len(handles)) {
				onReload()
			}
		}
	}()

	return func() {
		select {
		case <-done:
		default:
			close(done)
		}
		time.Sleep(50 * time.Millisecond)
	}, nil
}

func createOrOpen(eventName string) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(eventName)
	if err != nil {
		return 0, err
	}
	h, err := windows.CreateEvent(nil, 0, 0, name)
	if h != 0 {
		return h, nil
	}
	if err != nil {
		return windows.OpenEvent(windows.SYNCHRONIZE|windows.EVENT_MODIFY_STATE, false, name)
	}
	return 0, err
}

//go:build windows

package ipc

import (
	"testing"
	"time"
)

func TestRequestReloadWithoutWatcher(t *testing.T) {
	localEvent = `Local\WinMax_Reload_TestMissing`
	globalEvent = `Global\WinMax_Reload_TestMissing`
	t.Cleanup(func() {
		localEvent = `Local\WinMax_Reload`
		globalEvent = `Global\WinMax_Reload`
	})
	if err := RequestReload(); err == nil {
		t.Fatal("expected error when no watcher is running")
	}
}

func TestWatchAndRequestReload(t *testing.T) {
	localEvent = `Local\WinMax_Reload_Test`
	globalEvent = `Global\WinMax_Reload_Test`
	t.Cleanup(func() {
		localEvent = `Local\WinMax_Reload`
		globalEvent = `Global\WinMax_Reload`
	})

	got := make(chan struct{}, 1)
	stop, err := WatchReload(func() {
		select {
		case got <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	time.Sleep(50 * time.Millisecond)
	if err := RequestReload(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reload")
	}
}

//go:build windows

package singleinstance

import (
	"strings"
	"testing"
)

func TestAcquireExclusive(t *testing.T) {
	mutexName = `Local\WinMax_Mutex_Test`
	t.Cleanup(func() { mutexName = `Local\WinMax_UserLogonDaemon` })

	if Running() {
		t.Fatal("expected no instance before acquire")
	}

	first, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	if !Running() {
		t.Fatal("expected Running after acquire")
	}
	if _, err := Acquire(); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("second acquire = %v", err)
	}

	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	second.Close()
}

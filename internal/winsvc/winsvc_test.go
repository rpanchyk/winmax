//go:build windows

package winsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

func TestInstallRequiresElevation(t *testing.T) {
	if windows.GetCurrentProcessToken().IsElevated() {
		t.Skip("already running elevated")
	}
	err := Install(`C:\winmax.exe`)
	if err == nil || !strings.Contains(err.Error(), "elevated") {
		t.Fatalf("install without admin = %v", err)
	}
}

func TestUninstallRequiresElevation(t *testing.T) {
	if windows.GetCurrentProcessToken().IsElevated() {
		t.Skip("already running elevated")
	}
	err := Uninstall()
	if err == nil || !strings.Contains(err.Error(), "elevated") {
		t.Fatalf("uninstall without admin = %v", err)
	}
}

func TestStatusWithoutAdmin(t *testing.T) {
	text, err := Status()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "service:") {
		t.Fatalf("status = %q", text)
	}
}

func TestRequireConfigBeside(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "winmax.exe")
	if err := requireConfigBeside(exe); err == nil {
		t.Fatal("expected missing config error")
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte("apps: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := requireConfigBeside(exe); err != nil {
		t.Fatal(err)
	}
}

func TestEventWriter(t *testing.T) {
	w := eventWriter{}
	if n, err := w.Write([]byte("\n")); err != nil || n != 1 {
		t.Fatalf("empty write n=%d err=%v", n, err)
	}
}

func TestStateName(t *testing.T) {
	if stateName(svc.Stopped) != "stopped" || stateName(svc.Running) != "running" {
		t.Fatal("expected named states")
	}
	if stateName(999) != "state 999" {
		t.Fatalf("unknown state = %q", stateName(999))
	}
}

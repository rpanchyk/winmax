//go:build windows

package applog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathNextToExecutable(t *testing.T) {
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "winmax.log" {
		t.Fatalf("base = %q", filepath.Base(path))
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != filepath.Dir(exe) {
		t.Fatalf("log dir %q != exe dir %q", filepath.Dir(path), filepath.Dir(exe))
	}
}

func TestOpenWritesFile(t *testing.T) {
	logger, closer, err := Open(false)
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()
	logger.Print("test-line")

	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "test-line") {
		t.Fatalf("log file missing line: %q", data)
	}
}

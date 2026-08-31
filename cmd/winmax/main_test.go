//go:build windows

package main

import (
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{nil, {}, {"help"}, {"-h"}, {"--help"}} {
		if err := run(args); err != nil {
			t.Fatalf("run(%v) = %v", args, err)
		}
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := run([]string{"nope"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("run(nope) = %v", err)
	}
}

func TestReloadWithoutDaemon(t *testing.T) {
	err := run([]string{"reload"})
	if err == nil {
		t.Skip("a daemon is already running in this session")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Fatalf("reload without daemon = %v", err)
	}
}

func TestParseConfigFlag(t *testing.T) {
	path, err := parseConfigFlag([]string{"--config", `C:\cfg.yml`})
	if err != nil || path != `C:\cfg.yml` {
		t.Fatalf("got %q err=%v", path, err)
	}
	path, err = parseConfigFlag([]string{"-c", "x.yml"})
	if err != nil || path != "x.yml" {
		t.Fatalf("-c got %q err=%v", path, err)
	}
	path, err = parseConfigFlag([]string{"--config=y.yml"})
	if err != nil || path != "y.yml" {
		t.Fatalf("equals got %q err=%v", path, err)
	}
	if _, err := parseConfigFlag([]string{"--nope"}); err == nil {
		t.Fatal("expected unknown flag error")
	}
	if _, err := parseConfigFlag([]string{"--config"}); err == nil {
		t.Fatal("expected missing path error")
	}
}

func TestStatusCommand(t *testing.T) {
	if err := run([]string{"status"}); err != nil {
		t.Fatal(err)
	}
}

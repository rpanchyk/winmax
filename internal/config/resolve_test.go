package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := writeConfig(t, "apps: [")
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadEmptyApps(t *testing.T) {
	path := writeConfig(t, "apps: []\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected empty apps error")
	}
}

func TestLoadInvalidMatchShape(t *testing.T) {
	path := writeConfig(t, "apps:\n  - name: x\n    match: oops\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected match shape error")
	}
}

func TestLoadRepoConfig(t *testing.T) {
	path := filepath.Join("..", "..", "config.yml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Rules()) == 0 {
		t.Fatal("repo config.yml produced no rules")
	}
}

func TestResolvePathExplicit(t *testing.T) {
	path := writeConfig(t, `
apps:
  - name: "x"
    match:
      title: "x"
`)
	got, err := ResolvePath(path)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(path)
	if got != want {
		t.Fatalf("ResolvePath = %q, want %q", got, want)
	}
}

func TestResolvePathEnv(t *testing.T) {
	path := writeConfig(t, `
apps:
  - name: "env"
    match:
      process: "x.exe"
`)
	t.Setenv("WINMAX_CONFIG", path)
	got, err := ResolvePath("")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(path)
	if got != want {
		t.Fatalf("ResolvePath env = %q, want %q", got, want)
	}
}

func TestResolvePathMissing(t *testing.T) {
	t.Setenv("WINMAX_CONFIG", "")
	wd := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(wd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	t.Setenv("LOCALAPPDATA", filepath.Join(wd, "local"))
	_, err = ResolvePath("")
	if err == nil {
		t.Fatal("expected not found")
	}
	if !strings.Contains(err.Error(), "config.yml") && !os.IsNotExist(err) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRulesTrimsName(t *testing.T) {
	cfg := &Config{Apps: []App{{Name: "  MetaTrader  ", Match: Match{Title: "x"}}}}
	rules := cfg.Rules()
	if rules[0].Name != "MetaTrader" {
		t.Fatalf("name = %q", rules[0].Name)
	}
}

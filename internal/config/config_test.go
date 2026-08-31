package config

import (
	"os"
	"path/filepath"
	"testing"

	"winmax/internal/match"
)

func TestLoadMapMatch(t *testing.T) {
	path := writeConfig(t, `
apps:
  - name: "MetaTrader"
    match:
      condition: "AND"
      title: "MetaTrader"
      process: "terminal64.exe"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Apps[0].Name != "MetaTrader" || cfg.Apps[0].Match.Title != "MetaTrader" {
		t.Fatalf("unexpected app: %+v", cfg.Apps[0])
	}
	rule := cfg.Rules()[0]
	if rule.Name != "MetaTrader" || !rule.Match("MetaTrader 5", `C:\MT5\terminal64.exe`) {
		t.Fatalf("rules not applied: %+v", rule)
	}
}

func TestRepoConfigORMatchesBrokerTitle(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	rule := cfg.Rules()[0]
	if !rule.Match("52755108 - ICMarketsSC-Demo", `C:\Program Files\MetaTrader 5\terminal64.exe`) {
		t.Fatal("default OR config should match a broker title via process")
	}
	if !rule.Match("MetaTrader 4", `C:\MT4\terminal.exe`) {
		t.Fatal("default OR config should match MetaTrader 4 via title")
	}
}

func TestLoadListMatch(t *testing.T) {
	path := writeConfig(t, `
apps:
  - name: "MetaTrader"
    match:
      - condition: "OR"
      - title: "MetaTrader"
      - process: "terminal64.exe"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Apps[0].Match.Condition != "OR" || cfg.Apps[0].Match.Process != "terminal64.exe" {
		t.Fatalf("list match not merged: %+v", cfg.Apps[0].Match)
	}
	if !cfg.Rules()[0].Match("MetaTrader 4", `C:\MT4\terminal.exe`) {
		t.Fatal("OR list match should accept title-only")
	}
}

func TestValidate(t *testing.T) {
	if err := (&Config{Apps: []App{{Name: "x"}}}).validate(); err == nil {
		t.Fatal("expected missing match error")
	}
	if err := (&Config{Apps: []App{{Match: Match{Title: "x"}}}}).validate(); err == nil {
		t.Fatal("expected missing name error")
	}
	if err := (&Config{Apps: []App{{Name: "x", Match: Match{Title: "x", Condition: "XOR"}}}}).validate(); err == nil {
		t.Fatal("expected invalid condition error")
	}
	if err := (&Config{Apps: []App{{Name: "x", Match: Match{Title: "x"}}}}).validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultConditionIsAND(t *testing.T) {
	rule := match.Rule{Name: "x", Title: "MetaTrader", Process: "terminal64.exe"}
	if rule.Match("MetaTrader 5", `C:\MT5\terminal.exe`) {
		t.Fatal("default AND should require process too")
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

package match

import "testing"

func TestWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{"MetaTrader", "XAUUSD - MetaTrader 5", true},
		{"metatrader", "MetaTrader 5", true},
		{"*MetaTrader*", "MetaTrader 5", true},
		{"*MetaTrader*", "MetaTrader 4", true},
		{"MetaTrader*", "MetaTrader 5", true},
		{"MetaTrader*", "XAUUSD MetaTrader 5", false},
		{"*5", "MetaTrader 5", true},
		{"MetaTrader ?", "MetaTrader 5", true},
		{"MetaTrader ?", "MetaTrader 15", false},
		{"terminal64.exe", `C:\Program Files\MetaTrader 5\terminal64.exe`, true},
		{"*terminal64.exe*", `C:\Program Files\MetaTrader 5\terminal64.exe`, true},
		{"*terminal64.exe*", `C:\Program Files\MetaTrader 4\terminal64.exe`, true},
		{"*terminal64.exe*", `C:\Program Files\MetaTrader 4\terminal.exe`, false},
		{"", "MetaTrader", false},
		{"Notepad", "MetaTrader 5", false},
	}
	for _, tt := range tests {
		if got := wildcard(tt.pattern, tt.value); got != tt.want {
			t.Errorf("wildcard(%q, %q) = %v, want %v", tt.pattern, tt.value, got, tt.want)
		}
	}
}

func TestEmptyRuleNeverMatches(t *testing.T) {
	if (Rule{}).Match("MetaTrader 5", `C:\MT5\terminal64.exe`) {
		t.Fatal("empty rule should not match")
	}
}

func TestConditionAliases(t *testing.T) {
	or := Rule{Condition: " or ", Title: "MetaTrader", Process: "terminal64.exe"}
	if !or.Match("MetaTrader 4", `C:\MT4\terminal.exe`) {
		t.Fatal("lowercase OR should be accepted")
	}
	and := Rule{Condition: "", Title: "MetaTrader", Process: "terminal64.exe"}
	if and.Match("MetaTrader 4", `C:\MT4\terminal.exe`) {
		t.Fatal("empty condition should default to AND")
	}
}

func TestAnyAndString(t *testing.T) {
	rules := []Rule{
		{Name: "MetaTrader", Title: "MetaTrader", Process: "terminal64.exe"},
		{Name: "Notepad", Title: "Notepad"},
	}
	if !Any(rules, "Untitled - Notepad", `C:\Windows\notepad.exe`) {
		t.Fatal("expected Any to match Notepad")
	}
	if Any(rules, "Calculator", `C:\Windows\System32\calc.exe`) {
		t.Fatal("did not expect Any to match Calculator")
	}
	if got, ok := First(nil, "x", "y"); ok || got.Name != "" {
		t.Fatalf("First(nil) = %+v ok=%v", got, ok)
	}
	if s := (Rule{Name: "MetaTrader", Title: "T", Process: "P"}).String(); s != "MetaTrader [AND title=T process=P]" {
		t.Fatalf("String = %q", s)
	}
	if s := (Rule{}).String(); s != "(unnamed) [AND title= process=]" {
		t.Fatalf("unnamed String = %q", s)
	}
}

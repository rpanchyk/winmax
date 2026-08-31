package match

import "testing"

func TestANDMatch(t *testing.T) {
	mt := Rule{Name: "MetaTrader", Condition: "AND", Title: "MetaTrader", Process: "terminal64.exe"}
	tests := []struct {
		title string
		path  string
		want  bool
	}{
		{"XAUUSD,M1 - MetaTrader 5", `C:\Program Files\MetaTrader 5\terminal64.exe`, true},
		{"MetaTrader 4", `C:\Program Files\MetaTrader 4\terminal64.exe`, true},
		{"MetaTrader 5", `C:\Program Files\MetaTrader 5\terminal.exe`, false},
		{"Untitled - Notepad", `C:\Windows\notepad.exe`, false},
		{"", `C:\Program Files\MetaTrader 5\terminal64.exe`, false},
		{"MetaTrader 5", "", false},
	}
	for _, tt := range tests {
		if got := mt.Match(tt.title, tt.path); got != tt.want {
			t.Errorf("AND Match(%q, %q) = %v, want %v", tt.title, tt.path, got, tt.want)
		}
	}
}

func TestORMatch(t *testing.T) {
	r := Rule{Name: "MetaTrader", Condition: "OR", Title: "MetaTrader", Process: "terminal64.exe"}
	if !r.Match("MetaTrader 4", `C:\Program Files\MetaTrader 4\terminal.exe`) {
		t.Fatal("OR should match on title alone")
	}
	if !r.Match("IC Markets", `C:\Program Files\MetaTrader 5\terminal64.exe`) {
		t.Fatal("OR should match on process alone")
	}
	if r.Match("Notepad", `C:\Windows\notepad.exe`) {
		t.Fatal("OR should not match unrelated windows")
	}
}

func TestTitleOnly(t *testing.T) {
	r := Rule{Title: "*MetaTrader*"}
	if !r.Match("MetaTrader 5", `C:\Windows\notepad.exe`) {
		t.Fatal("title wildcard should match regardless of process")
	}
	if r.Match("Calculator", `C:\MT5\terminal64.exe`) {
		t.Fatal("did not expect a title match")
	}
}

func TestProcessOnly(t *testing.T) {
	r := Rule{Process: "*terminal64.exe*"}
	if !r.Match("", `C:\Program Files\MetaTrader 5\terminal64.exe`) {
		t.Fatal("process wildcard should match the full path")
	}
	if r.Match("MetaTrader 4", `C:\Program Files\MetaTrader 4\terminal.exe`) {
		t.Fatal("did not expect terminal.exe to match terminal64.exe")
	}
}

func TestFirstReturnsName(t *testing.T) {
	rules := []Rule{
		{Name: "MetaTrader", Title: "MetaTrader", Process: "terminal64.exe"},
		{Name: "Notepad", Title: "Notepad"},
	}
	got, ok := First(rules, "Untitled - Notepad", `C:\Windows\notepad.exe`)
	if !ok || got.Name != "Notepad" {
		t.Fatalf("First = %+v, ok=%v", got, ok)
	}
}

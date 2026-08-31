//go:build windows

package autostart

import "testing"

func TestQuote(t *testing.T) {
	if got := quote(`C:\Tools\winmax.exe`); got != `"C:\Tools\winmax.exe"` {
		t.Fatalf("quote = %q", got)
	}
	if got := quote(`"C:\Tools\winmax.exe"`); got != `"C:\Tools\winmax.exe"` {
		t.Fatalf("already quoted = %q", got)
	}
}

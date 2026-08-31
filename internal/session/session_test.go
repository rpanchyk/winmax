//go:build windows

package session

import (
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestNextBackoff(t *testing.T) {
	if got := nextBackoff(0); got != minBackoff {
		t.Fatalf("from zero = %s", got)
	}
	if got := nextBackoff(minBackoff); got != 4*time.Second {
		t.Fatalf("from min = %s", got)
	}
	if got := nextBackoff(maxBackoff); got != maxBackoff {
		t.Fatalf("at max = %s", got)
	}
}

func TestHasInteractiveSession(t *testing.T) {
	_ = HasInteractiveSession()
}

func TestSessionWanted(t *testing.T) {
	if !sessionWanted(windows.WTSActive) || !sessionWanted(windows.WTSConnected) || !sessionWanted(windows.WTSDisconnected) {
		t.Fatal("expected logged-on session states")
	}
	if sessionWanted(windows.WTSListen) || sessionWanted(windows.WTSIdle) {
		t.Fatal("did not expect idle/listen sessions")
	}
}

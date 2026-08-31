//go:build windows

package daemon

import (
	"io"
	"log"
	"testing"

	"winmax/internal/match"
)

func TestNewAndSetRules(t *testing.T) {
	d := New([]match.Rule{{Name: "A", Title: "A"}}, log.New(io.Discard, "", 0))
	got := d.snapshotRules()
	if len(got) != 1 || got[0].Name != "A" {
		t.Fatalf("initial rules: %+v", got)
	}

	key := handledKey{hwnd: 1, pid: 2}
	d.markHandled(key)
	if !d.alreadyHandled(key) {
		t.Fatal("expected key to be handled")
	}

	d.SetRules([]match.Rule{{Name: "B", Title: "B"}, {Name: "C", Process: "c.exe"}})
	got = d.snapshotRules()
	if len(got) != 2 || got[0].Name != "B" || got[1].Name != "C" {
		t.Fatalf("reloaded rules: %+v", got)
	}
	if d.alreadyHandled(key) {
		t.Fatal("SetRules should clear handled windows")
	}
}

func TestForgetHandled(t *testing.T) {
	d := New(nil, log.New(io.Discard, "", 0))
	d.markHandled(handledKey{hwnd: 10, pid: 1})
	d.markHandled(handledKey{hwnd: 10, pid: 2})
	d.markHandled(handledKey{hwnd: 11, pid: 3})
	d.forget(10)
	if d.alreadyHandled(handledKey{hwnd: 10, pid: 1}) || d.alreadyHandled(handledKey{hwnd: 10, pid: 2}) {
		t.Fatal("forget should drop every pid for the hwnd")
	}
	if !d.alreadyHandled(handledKey{hwnd: 11, pid: 3}) {
		t.Fatal("forget should not drop other hwnds")
	}
}

func TestSnapshotIsCopy(t *testing.T) {
	d := New([]match.Rule{{Name: "A"}}, log.New(io.Discard, "", 0))
	got := d.snapshotRules()
	got[0].Name = "mutated"
	if d.snapshotRules()[0].Name != "A" {
		t.Fatal("snapshotRules should return a copy")
	}
}

func TestClaimIsExclusive(t *testing.T) {
	d := New(nil, log.New(io.Discard, "", 0))
	if !d.claim(42) {
		t.Fatal("first claim should succeed")
	}
	if d.claim(42) {
		t.Fatal("second claim for the same hwnd should fail")
	}
	if !d.claim(43) {
		t.Fatal("a different hwnd should still be claimable")
	}
	d.release(42)
	if !d.claim(42) {
		t.Fatal("claim should work again after release")
	}
}

func TestWinEventProcIgnoresNoise(t *testing.T) {
	if winEventProc(0, EVENT_SHOW, 0, 0, 0, 0, 0) != 0 {
		t.Fatal("null hwnd should be ignored")
	}
	if winEventProc(0, EVENT_SHOW, 1, 1, 0, 0, 0) != 0 {
		t.Fatal("non-window object should be ignored")
	}
}

const EVENT_SHOW = 0x8002

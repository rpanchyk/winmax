//go:build windows

package win32

import (
	"os"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestNullWindowQueries(t *testing.T) {
	if IsWindow(0) {
		t.Fatal("null hwnd is not a window")
	}
	if IsWindowVisible(0) {
		t.Fatal("null hwnd is not visible")
	}
	if IsZoomed(0) {
		t.Fatal("null hwnd is not zoomed")
	}
	if GetWindowText(0) != "" {
		t.Fatal("null hwnd should have an empty title")
	}
	if GetWindowProcessID(0) != 0 {
		t.Fatal("null hwnd should have pid 0")
	}
	if _, ok := GetWindowRect(0); ok {
		t.Fatal("null hwnd should not have a rect")
	}
	if PlacementMaximized(0) {
		t.Fatal("null hwnd is not placement-maximized")
	}
	if IsOwned(0) {
		t.Fatal("null hwnd is not owned")
	}
	if IsDialogWindow(0) {
		t.Fatal("null hwnd is not a dialog")
	}
}

func TestOwnsConsoleDoesNotPanic(t *testing.T) {
	_ = OwnsConsole()
}

func TestProcessImagePathCurrent(t *testing.T) {
	pid := uint32(os.Getpid())
	path := ProcessImagePath(pid)
	if path == "" {
		t.Fatal("expected a path or exe name for the test process")
	}
}

func TestProcessImageForWindowNull(t *testing.T) {
	if ProcessImageForWindow(0, 0) != "" {
		t.Fatal("null window should not resolve a path")
	}
}

func TestMaximizeRealWindow(t *testing.T) {
	hwnd, cleanup, err := createTestWindow("WinMaxMaximizeTest")
	if err != nil {
		t.Skip(err)
	}
	defer cleanup()

	ShowWindow(hwnd, 5) // SW_SHOW
	if IsZoomed(hwnd) {
		t.Fatal("fresh window should not start maximized")
	}
	ShowWindow(hwnd, SW_MAXIMIZE)
	if !IsZoomed(hwnd) {
		t.Fatal("ShowWindow(SW_MAXIMIZE) did not maximize the window")
	}
	if !PlacementMaximized(hwnd) {
		t.Fatal("maximized window should report SW_SHOWMAXIMIZED placement")
	}
}

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   windows.Handle
	icon       windows.Handle
	cursor     windows.Handle
	background windows.Handle
	menuName   *uint16
	className  *uint16
	iconSm     windows.Handle
}

func TestOwnedDialogIsSkipped(t *testing.T) {
	parent, cleanupParent, err := createTestWindow("WinMaxOwnerParent")
	if err != nil {
		t.Skip(err)
	}
	defer cleanupParent()
	ShowWindow(parent, 5)

	owned, cleanupOwned, err := createTestWindowEx("WinMaxOwnedDlg", 0, parent)
	if err != nil {
		t.Skip(err)
	}
	defer cleanupOwned()
	ShowWindow(owned, 5)

	dialog, cleanupDialog, err := createTestWindowEx("WinMaxDlgFrame", WS_EX_DLGMODALFRAME, 0)
	if err != nil {
		t.Skip(err)
	}
	defer cleanupDialog()
	ShowWindow(dialog, 5)

	if IsOwned(parent) || IsDialogWindow(parent) {
		t.Fatal("main window should not be treated as a dialog")
	}
	if !IsTopLevel(owned) {
		t.Fatal("owned window is still top-level, so IsTopLevel is not enough")
	}
	if !IsOwned(owned) || !IsDialogWindow(owned) {
		t.Fatal("owned window should be treated as a dialog")
	}
	if Owner(owned) != parent {
		t.Fatalf("owner=%#x want %#x", Owner(owned), parent)
	}
	if !IsDialogWindow(dialog) {
		t.Fatal("WS_EX_DLGMODALFRAME should be treated as a dialog")
	}
	if !IsWindowEnabled(parent) {
		t.Fatal("parent should start enabled")
	}

	enable := user32.NewProc("EnableWindow")
	enable.Call(parent, 0)
	if IsWindowEnabled(parent) {
		t.Fatal("EnableWindow(false) should disable the parent")
	}
	enable.Call(parent, 1)
}

func createTestWindow(class string) (uintptr, func(), error) {
	return createTestWindowEx(class, 0, 0)
}

func createTestWindowEx(class string, exStyle, owner uintptr) (uintptr, func(), error) {
	className, err := windows.UTF16PtrFromString(class)
	if err != nil {
		return 0, nil, err
	}
	title, err := windows.UTF16PtrFromString("WinMax test")
	if err != nil {
		return 0, nil, err
	}

	mod, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	defProc := user32.NewProc("DefWindowProcW")
	wndProc := syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
		r, _, _ := defProc.Call(hwnd, msg, wParam, lParam)
		return r
	})

	wc := wndClassEx{
		wndProc:   wndProc,
		instance:  windows.Handle(mod),
		className: className,
	}
	wc.size = uint32(unsafe.Sizeof(wc))
	atom, _, err := user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return 0, nil, err
	}

	const (
		wsOverlappedWindow = 0x00CF0000
		cwUseDefault       = 0x80000000
	)
	hwnd, _, err := user32.NewProc("CreateWindowExW").Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow,
		cwUseDefault, cwUseDefault, 400, 300,
		owner, 0, mod, 0,
	)
	if hwnd == 0 {
		user32.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(className)), mod)
		return 0, nil, err
	}

	cleanup := func() {
		user32.NewProc("DestroyWindow").Call(hwnd)
		user32.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(className)), mod)
	}
	return hwnd, cleanup, nil
}

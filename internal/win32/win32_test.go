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

func createTestWindow(class string) (uintptr, func(), error) {
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
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow,
		cwUseDefault, cwUseDefault, 400, 300,
		0, 0, mod, 0,
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

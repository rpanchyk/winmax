//go:build windows

package win32

import (
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	EVENT_OBJECT_DESTROY    = 0x8001
	EVENT_OBJECT_SHOW       = 0x8002
	EVENT_OBJECT_NAMECHANGE = 0x800C

	WINEVENT_OUTOFCONTEXT   = 0x0000
	WINEVENT_SKIPOWNPROCESS = 0x0002

	OBJID_WINDOW = 0

	SW_HIDE     = 0
	SW_MAXIMIZE = 3

	GWL_STYLE   = -16
	GWL_EXSTYLE = -20

	WS_MAXIMIZEBOX = 0x00010000
	WS_CHILD       = 0x40000000

	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_APPWINDOW  = 0x00040000

	GA_ROOT = 2

	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

	WM_QUIT = 0x0012
)

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procSetWinEventHook            = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent             = user32.NewProc("UnhookWinEvent")
	procGetMessageW                = user32.NewProc("GetMessageW")
	procTranslateMessage           = user32.NewProc("TranslateMessage")
	procDispatchMessageW           = user32.NewProc("DispatchMessageW")
	procShowWindow                 = user32.NewProc("ShowWindow")
	procIsWindow                   = user32.NewProc("IsWindow")
	procIsWindowVisible            = user32.NewProc("IsWindowVisible")
	procIsZoomed                   = user32.NewProc("IsZoomed")
	procGetWindowTextW             = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW       = user32.NewProc("GetWindowTextLengthW")
	procGetWindowLongPtrW          = user32.NewProc("GetWindowLongPtrW")
	procGetWindowThreadProcessId   = user32.NewProc("GetWindowThreadProcessId")
	procGetAncestor                = user32.NewProc("GetAncestor")
	procEnumWindows                = user32.NewProc("EnumWindows")
	procGetConsoleWindow           = kernel32.NewProc("GetConsoleWindow")
	procGetConsoleProcessList      = kernel32.NewProc("GetConsoleProcessList")
	procFreeConsole                = kernel32.NewProc("FreeConsole")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procGetWindowModuleFileNameW   = user32.NewProc("GetWindowModuleFileNameW")
	procPostThreadMessageW         = user32.NewProc("PostThreadMessageW")
)

func SetWinEventHook(eventMin, eventMax uint32, cb uintptr) (uintptr, error) {
	r, _, err := procSetWinEventHook.Call(
		uintptr(eventMin),
		uintptr(eventMax),
		0,
		cb,
		0,
		0,
		WINEVENT_OUTOFCONTEXT|WINEVENT_SKIPOWNPROCESS,
	)
	if r == 0 {
		return 0, err
	}
	return r, nil
}

func UnhookWinEvent(hook uintptr) {
	procUnhookWinEvent.Call(hook)
}

func GetMessage(msg *MSG) int32 {
	r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(msg)), 0, 0, 0)
	return int32(r)
}

func TranslateMessage(msg *MSG) {
	procTranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
}

func DispatchMessage(msg *MSG) {
	procDispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
}

func ShowWindow(hwnd uintptr, cmd int) bool {
	r, _, _ := procShowWindow.Call(hwnd, uintptr(cmd))
	return r != 0
}

func IsWindow(hwnd uintptr) bool {
	r, _, _ := procIsWindow.Call(hwnd)
	return r != 0
}

func IsWindowVisible(hwnd uintptr) bool {
	r, _, _ := procIsWindowVisible.Call(hwnd)
	return r != 0
}

func IsZoomed(hwnd uintptr) bool {
	r, _, _ := procIsZoomed.Call(hwnd)
	return r != 0
}

func GetWindowText(hwnd uintptr) string {
	n, _, _ := procGetWindowTextLengthW.Call(hwnd)
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(n+1))
	return syscall.UTF16ToString(buf)
}

func GetWindowLongPtr(hwnd uintptr, index int32) uintptr {
	r, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(index))
	return r
}

func GetWindowProcessID(hwnd uintptr) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

func GetAncestor(hwnd uintptr, flags uint32) uintptr {
	r, _, _ := procGetAncestor.Call(hwnd, uintptr(flags))
	return r
}

func EnumWindows(cb uintptr) {
	procEnumWindows.Call(cb, 0)
}

func ProcessImageForWindow(hwnd uintptr, pid uint32) string {
	if path := ProcessImagePath(pid); path != "" {
		return path
	}
	return WindowModuleFileName(hwnd)
}

func ProcessImagePath(pid uint32) string {
	if pid == 0 {
		return ""
	}
	if path := queryFullProcessImageName(pid); path != "" {
		return path
	}
	return processExeFromSnapshot(pid)
}

func queryFullProcessImageName(pid uint32) string {
	h, err := windows.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)

	size := uint32(32768)
	buf := make([]uint16, size)
	r, _, _ := procQueryFullProcessImageNameW.Call(
		uintptr(h),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}

func WindowModuleFileName(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	var buf [32768]uint16
	n, _, _ := procGetWindowModuleFileNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

func processExeFromSnapshot(pid uint32) string {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(snap)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snap, &entry); err != nil {
		return ""
	}
	for {
		if entry.ProcessID == pid {
			return windows.UTF16ToString(entry.ExeFile[:])
		}
		if err := windows.Process32Next(snap, &entry); err != nil {
			return ""
		}
	}
}

func ProcessBaseName(pid uint32) string {
	path := ProcessImagePath(pid)
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

func PostThreadQuit(threadID uint32) {
	procPostThreadMessageW.Call(uintptr(threadID), WM_QUIT, 0, 0)
}

func OwnsConsole() bool {
	return ownsConsole()
}

func HideConsole() {
	if !ownsConsole() {
		return
	}
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd == 0 {
		return
	}
	procShowWindow.Call(hwnd, SW_HIDE)
	procFreeConsole.Call()
}

func ownsConsole() bool {
	var pids [8]uint32
	n, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	return n == 1
}

func IsTopLevel(hwnd uintptr) bool {
	style := GetWindowLongPtr(hwnd, GWL_STYLE)
	if style&WS_CHILD != 0 {
		return false
	}
	return GetAncestor(hwnd, GA_ROOT) == hwnd
}

func IsToolWindow(hwnd uintptr) bool {
	ex := GetWindowLongPtr(hwnd, GWL_EXSTYLE)
	if ex&WS_EX_TOOLWINDOW == 0 {
		return false
	}
	return ex&WS_EX_APPWINDOW == 0
}

func CanMaximize(hwnd uintptr) bool {
	style := GetWindowLongPtr(hwnd, GWL_STYLE)
	return style&WS_MAXIMIZEBOX != 0
}

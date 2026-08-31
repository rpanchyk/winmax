//go:build windows

package daemon

import (
	"log"
	"sync"
	"syscall"
	"time"

	"winmax/internal/match"
	"winmax/internal/win32"
)

const (
	retryAttempts = 5
	retryDelay    = 200 * time.Millisecond
)

type handledKey struct {
	hwnd uintptr
	pid  uint32
}

type Daemon struct {
	rules []match.Rule
	log   *log.Logger

	mu       sync.Mutex
	handled  map[handledKey]struct{}
	inflight map[uintptr]struct{}

	hooks []uintptr
}

func New(rules []match.Rule, logger *log.Logger) *Daemon {
	if logger == nil {
		logger = log.Default()
	}
	return &Daemon{
		rules:    rules,
		log:      logger,
		handled:  make(map[handledKey]struct{}),
		inflight: make(map[uintptr]struct{}),
	}
}

var (
	active     *Daemon
	eventCB    = syscall.NewCallback(winEventProc)
	enumCB     = syscall.NewCallback(enumWindowsProc)
	activeLock sync.Mutex
)

func (d *Daemon) Run() error {
	activeLock.Lock()
	active = d
	activeLock.Unlock()
	defer func() {
		activeLock.Lock()
		active = nil
		activeLock.Unlock()
	}()

	for _, ev := range []uint32{win32.EVENT_OBJECT_SHOW, win32.EVENT_OBJECT_NAMECHANGE, win32.EVENT_OBJECT_DESTROY} {
		hook, err := win32.SetWinEventHook(ev, ev, eventCB)
		if err != nil {
			d.unhook()
			return err
		}
		d.hooks = append(d.hooks, hook)
	}

	d.log.Printf("watching new windows for: %v", d.snapshotRules())
	win32.EnumWindows(enumCB)

	var msg win32.MSG
	for {
		ret := win32.GetMessage(&msg)
		if ret <= 0 {
			d.unhook()
			if ret < 0 {
				return syscall.EINVAL
			}
			return nil
		}
		win32.TranslateMessage(&msg)
		win32.DispatchMessage(&msg)
	}
}

func (d *Daemon) unhook() {
	for _, hook := range d.hooks {
		win32.UnhookWinEvent(hook)
	}
	d.hooks = nil
}

func winEventProc(_, event, hwnd uintptr, idObject, _, _, _ uintptr) uintptr {
	if hwnd == 0 || int32(idObject) != win32.OBJID_WINDOW {
		return 0
	}
	activeLock.Lock()
	d := active
	activeLock.Unlock()
	if d == nil {
		return 0
	}

	switch event {
	case win32.EVENT_OBJECT_DESTROY:
		d.forget(hwnd)
	case win32.EVENT_OBJECT_SHOW, win32.EVENT_OBJECT_NAMECHANGE:
		go d.consider(hwnd)
	}
	return 0
}

func enumWindowsProc(hwnd, _ uintptr) uintptr {
	activeLock.Lock()
	d := active
	activeLock.Unlock()
	if d != nil {
		d.consider(hwnd)
	}
	return 1
}

func (d *Daemon) consider(hwnd uintptr) {
	if !d.claim(hwnd) {
		return
	}
	defer d.release(hwnd)

	for i := 0; i < retryAttempts; i++ {
		if d.tryMaximize(hwnd) {
			return
		}
		time.Sleep(retryDelay)
	}
}

func (d *Daemon) tryMaximize(hwnd uintptr) bool {
	if !win32.IsWindow(hwnd) || !win32.IsTopLevel(hwnd) || !win32.IsWindowVisible(hwnd) {
		return true
	}
	if win32.IsToolWindow(hwnd) || !win32.CanMaximize(hwnd) {
		return true
	}

	pid := win32.GetWindowProcessID(hwnd)
	key := handledKey{hwnd: hwnd, pid: pid}
	if d.alreadyHandled(key) {
		return true
	}

	title := win32.GetWindowText(hwnd)
	path := win32.ProcessImageForWindow(hwnd, pid)
	rule, ok := match.First(d.snapshotRules(), title, path)
	if !ok {
		// Title is often empty on the first SHOW; keep retrying until it appears.
		return title != ""
	}
	if win32.IsZoomed(hwnd) {
		d.markHandled(key)
		return true
	}

	// ShowWindow's return is "was visible", not success. Call it once; a retry
	// would maximize the same window again (SHOW also fires another event).
	d.markHandled(key)
	win32.ShowWindow(hwnd, win32.SW_MAXIMIZE)
	d.log.Printf("maximized app=%s hwnd=%#x pid=%d title=%q exe=%s", rule.Name, hwnd, pid, title, path)
	return true
}

func (d *Daemon) SetRules(rules []match.Rule) {
	d.mu.Lock()
	d.rules = rules
	d.handled = make(map[handledKey]struct{})
	d.inflight = make(map[uintptr]struct{})
	d.mu.Unlock()
	d.log.Printf("reloaded rules: %v", rules)
	win32.EnumWindows(enumCB)
}

func (d *Daemon) snapshotRules() []match.Rule {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]match.Rule, len(d.rules))
	copy(out, d.rules)
	return out
}

func (d *Daemon) claim(hwnd uintptr) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.inflight[hwnd]; ok {
		return false
	}
	d.inflight[hwnd] = struct{}{}
	return true
}

func (d *Daemon) release(hwnd uintptr) {
	d.mu.Lock()
	delete(d.inflight, hwnd)
	d.mu.Unlock()
}

func (d *Daemon) alreadyHandled(key handledKey) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.handled[key]
	return ok
}

func (d *Daemon) markHandled(key handledKey) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handled[key] = struct{}{}
}

func (d *Daemon) forget(hwnd uintptr) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.inflight, hwnd)
	for key := range d.handled {
		if key.hwnd == hwnd {
			delete(d.handled, key)
		}
	}
}

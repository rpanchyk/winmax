//go:build windows

package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"

	"winmax/internal/applog"
	"winmax/internal/config"
	"winmax/internal/daemon"
	"winmax/internal/ipc"
	"winmax/internal/singleinstance"
	"winmax/internal/win32"
	"winmax/internal/winsvc"
)

func main() {
	inService, err := svc.IsWindowsService()
	if err == nil && inService {
		if err := winsvc.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "winmax: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "winmax: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := ""
	if len(args) > 0 {
		cmd = strings.ToLower(args[0])
	}

	switch cmd {
	case "help", "-h", "--help", "":
		printUsage()
		return nil
	case "install":
		return install()
	case "uninstall":
		return uninstall()
	case "reload":
		return reload()
	case "status":
		return status()
	case "console", "foreground":
		cfgPath, err := parseConfigFlag(args[1:])
		if err != nil {
			return err
		}
		return runWatcher(true, cfgPath)
	case "worker":
		cfgPath, err := parseConfigFlag(args[1:])
		if err != nil {
			return err
		}
		return runWatcher(false, cfgPath)
	default:
		return fmt.Errorf("unknown command %q\n\nRun 'winmax help' for usage.", cmd)
	}
}

func parseConfigFlag(args []string) (string, error) {
	path := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--config" || a == "-c":
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a path", a)
			}
			i++
			path = args[i]
		case strings.HasPrefix(a, "--config="):
			path = strings.TrimPrefix(a, "--config=")
		default:
			return "", fmt.Errorf("unknown flag %q", a)
		}
	}
	return path, nil
}

func printUsage() {
	fmt.Print(`winmax — maximize new windows by config.yml rules

Usage:
  winmax help         show this help
  winmax install      install as a Windows service (requires Administrator)
  winmax uninstall    stop and remove the Windows service (requires Administrator)
  winmax reload       reload config.yml in the running daemon
  winmax foreground   run in the foreground with logs (Ctrl+C to stop)
  winmax console      run in the console with logs (Ctrl+C to stop)
  winmax status       show Windows service status

  --config, -c PATH   config file for console / foreground / worker
`)
}

func runWatcher(console bool, explicitConfig string) error {
	runtime.LockOSThread()

	logger, closer, err := applog.Open(console)
	if err != nil {
		writeWorkerError(err)
		return err
	}
	defer closer.Close()

	cfgPath, err := config.ResolvePath(explicitConfig)
	if err != nil {
		writeWorkerError(err)
		return err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		writeWorkerError(err)
		return err
	}
	logger.Printf("loaded config %s", cfgPath)

	handle, err := singleinstance.Acquire()
	if err != nil {
		writeWorkerError(err)
		return err
	}
	defer handle.Close()

	if !console {
		win32.HideConsole()
	}

	d := daemon.New(cfg.Rules(), logger)
	stopReload, err := ipc.WatchReload(func() {
		fresh, loadErr := config.Load(cfgPath)
		if loadErr != nil {
			logger.Printf("reload failed: %v", loadErr)
			return
		}
		d.SetRules(fresh.Rules())
		logger.Printf("reloaded config %s", cfgPath)
	})
	if err != nil {
		return err
	}
	defer stopReload()

	tid := windows.GetCurrentThreadId()
	go quitOnSignal(tid)

	return d.Run()
}

func writeWorkerError(err error) {
	path, pErr := applog.Path()
	if pErr != nil {
		return
	}
	fallback := filepath.Join(filepath.Dir(path), "winmax-worker.err")
	line := time.Now().Format(time.RFC3339) + " " + err.Error() + "\n"
	f, oErr := os.OpenFile(fallback, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if oErr != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func install() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return err
	}
	if err := winsvc.Install(exe); err != nil {
		return err
	}
	fmt.Printf("installed Windows service %s\n", winsvc.Name)
	return nil
}

func uninstall() error {
	if err := winsvc.Uninstall(); err != nil {
		return err
	}
	fmt.Printf("removed Windows service %s\n", winsvc.Name)
	return nil
}

func reload() error {
	if err := ipc.RequestReload(); err != nil {
		return err
	}
	fmt.Println("reload requested")
	return nil
}

func status() error {
	text, err := winsvc.Status()
	if err != nil {
		return err
	}
	fmt.Println(text)
	return nil
}

func quitOnSignal(threadID uint32) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
	win32.PostThreadQuit(threadID)
}

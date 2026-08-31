//go:build windows

package autostart

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName = "WinMax"
)

func Install(exePath string) error {
	if exePath == "" {
		return fmt.Errorf("executable path is empty")
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer key.Close()

	value := quote(exePath)
	if err := key.SetStringValue(valueName, value); err != nil {
		return fmt.Errorf("set Run value: %w", err)
	}
	return nil
}

func Uninstall() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err == registry.ErrNotExist {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer key.Close()

	if err := key.DeleteValue(valueName); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("delete Run value: %w", err)
	}
	return nil
}

func Enabled() (string, bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err == registry.ErrNotExist {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("open Run key: %w", err)
	}
	defer key.Close()

	value, _, err := key.GetStringValue(valueName)
	if err == registry.ErrNotExist {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read Run value: %w", err)
	}
	return value, true, nil
}

func quote(path string) string {
	if strings.HasPrefix(path, `"`) {
		return path
	}
	return `"` + path + `"`
}

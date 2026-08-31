//go:build windows

package applog

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

func Path() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "winmax.log"), nil
}

func Open(foreground bool) (*log.Logger, io.Closer, error) {
	path, err := Path()
	if err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	var w io.Writer = f
	if foreground {
		w = io.MultiWriter(os.Stdout, f)
	}
	return log.New(w, "winmax: ", log.LstdFlags), f, nil
}

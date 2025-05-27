//go:build windows
// +build windows

package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func GetSystemShell() string {
	if shell, err := exec.LookPath("pwsh"); err == nil {
		return shell
	} else if shell, err := exec.LookPath("powershell"); err == nil {
		return shell
	} else {
		fmt.Fprint(os.Stderr, "Error finding system shell\r\n")
		os.Exit(1)
		return ""
	}
}

func defaultConfigDir() string {
	return filepath.Join(os.Getenv("APPDATA"), "ohmygosh")
}

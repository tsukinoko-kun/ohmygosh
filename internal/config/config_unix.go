//go:build !windows
// +build !windows

package config

import (
	"fmt"
	"os"
	"os/exec"
)

func GetSystemShell() string {
	shell, ok := os.LookupEnv("SHELL")
	if !ok {
		if bash, err := exec.LookPath("bash"); err == nil {
			shell = bash
		} else if zsh, err := exec.LookPath("zsh"); err == nil {
			shell = zsh
		} else if sh, err := exec.LookPath("sh"); err == nil {
			shell = sh
		} else {
			fmt.Fprintf(os.Stderr, "Error finding system shell: %v\n", err)
			os.Exit(1)
		}
	}
	return shell
}

//go:build windows
// +build windows

package data

import (
	"os"
	"path/filepath"
)

func noXdgDataHome() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "ohmygosh")
}

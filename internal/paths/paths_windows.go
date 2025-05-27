//go:build windows
// +build windows

package paths

import (
	"strings"
)

func Unixify(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

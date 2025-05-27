//go:build !windows
// +build !windows

package paths

func Unixify(path string) string {
	return path
}

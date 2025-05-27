//go:build windows
// +build windows

package shell

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/tsukinoko-kun/ohmygosh/internal/config"
)

func GetShellArgv() (string, []string) {
	shell := config.Get.Shell.Exe
	args := make([]string, len(config.Get.Shell.Args), len(config.Get.Shell.Args)+1)
	copy(args, config.Get.Shell.Args)
	args = append(args, "-EncodedCommand")
	return shell, args
}

func Aliases() string {
	aliases := make(map[string]string)
	sb := strings.Builder{}
	aliases["close"] = fmt.Sprintf(`Invoke-RestMethod -Uri "%s/ipc" -Method POST -Headers @{"X-Key" = %q} -Body "exit $($args[0])"`, ipcAddr, ipcKey)
	for alias, cmd := range aliases {
		sb.WriteString(fmt.Sprintf(`function %s { %s } ; `, alias, cmd))
	}
	return sb.String()
}

func Wrap(cmd string) string {
	return fmt.Sprintf(`try { %s } finally { Invoke-RestMethod -Uri "%s/ipc" -Method POST -Headers @{"X-Key" = %q} -Body "cd $(pwd)" }`, cmd, ipcAddr, ipcKey)
}

// Escape provides an alternative approach using
// Base64 encoding, which is more reliable for complex commands with
// many special characters. Use with powershell.exe -EncodedCommand
func Escape(command string) string {
	// Convert to UTF-16LE (PowerShell's expected encoding for -EncodedCommand)
	utf16Bytes := utf16.Encode([]rune(command))
	bytes := make([]byte, len(utf16Bytes)*2)

	for i, r := range utf16Bytes {
		bytes[i*2] = byte(r)
		bytes[i*2+1] = byte(r >> 8)
	}

	return base64.StdEncoding.EncodeToString(bytes)
}

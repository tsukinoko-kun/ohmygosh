//go:build !windows
// +build !windows

package shell

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/tsukinoko-kun/ohmygosh/internal/config"
)

func GetShellArgv() (string, []string) {
	shell := config.Get.Shell.Exe
	args := make([]string, len(config.Get.Shell.Args), len(config.Get.Shell.Args)+1)
	copy(args, config.Get.Shell.Args)
	args = append(args, "-c")
	return shell, args
}

func Aliases() string {
	aliases := make(map[string]string)
	sb := strings.Builder{}
	if _, err := exec.LookPath("curl"); err == nil {
		aliases["exit"] = fmt.Sprintf(`curl -X POST -H "X-Key: %s" %s/ipc -d "exit $1" ; builtin exit $1`, ipcKey, ipcAddr)
		aliases["close"] = fmt.Sprintf(`curl -X POST -H "X-Key: %s" %s/ipc -d "exit $1" ; builtin exit $1`, ipcKey, ipcAddr)
	} else if _, err := exec.LookPath("wget"); err == nil {
		aliases["exit"] = fmt.Sprintf(`wget --method=POST --header="X-Key: %s" --post-data="exit $1" %s/ipc -O - ; builtin exit $1`, ipcKey, ipcAddr)
		aliases["close"] = fmt.Sprintf(`wget --method=POST --header="X-Key: %s" --post-data="exit $1" %s/ipc -O - ; builtin exit $1`, ipcKey, ipcAddr)
	}
	for alias, cmd := range aliases {
		sb.WriteString(fmt.Sprintf(`%s() { %s } ; `, alias, cmd))
	}
	return sb.String()
}

func Wrap(cmd string) string {
	if _, err := exec.LookPath("curl"); err == nil {
		return fmt.Sprintf(`%s ; ohmybashexitcode=$? ; curl -X POST -H "X-Key: %s" %s/ipc -d "cd $(pwd)" ; builtin exit $ohmybashexitcode`, cmd, ipcKey, ipcAddr)
	} else if _, err := exec.LookPath("wget"); err == nil {
		return fmt.Sprintf(`%s ; ohmybashexitcode=$? ; wget --method=POST --header="X-Key: %s" --post-data="cd $(pwd)" %s/ipc -O - ; builtin exit $ohmybashexitcode`, cmd, ipcKey, ipcAddr)
	} else {
		return cmd
	}
}

func Escape(s string) string {
	return s
}

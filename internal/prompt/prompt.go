package prompt

import (
	"os"
	"strings"

	"github.com/tsukinoko-kun/ohmygosh/internal/shell"
)

var userHomeDir string

func init() {
	userHomeDir = os.Getenv("HOME")
}

func Get() string {
	return strings.Replace(shell.Wd, userHomeDir, "~", 1) + gitBranch()
}

func Boring() string {
	return strings.Replace(shell.Wd, userHomeDir, "~", 1)
}

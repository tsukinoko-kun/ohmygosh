package exit

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
)

var ExitCode int

var P *tea.Program

var size *pty.Winsize

type Command struct {
	Cmd *exec.Cmd
	Pty *os.File
}

var TrackedCommands []*Command

func Exit(code int) {
	ExitCode = code
	P.Quit()
}

func TrackCommand(cmd *exec.Cmd, t *os.File) {
	TrackedCommands = append(TrackedCommands, &Command{Cmd: cmd, Pty: t})
	if size != nil && t != nil {
		_ = pty.Setsize(t, size)
	}
}

func ClearTrackedCommands() {
	TrackedCommands = nil
}

func InheritSize(size *pty.Winsize) error {
	for _, cmd := range TrackedCommands {
		if cmd.Pty != nil {
			if err := pty.Setsize(cmd.Pty, size); err != nil {
				return err
			}
		}
	}
	return nil
}

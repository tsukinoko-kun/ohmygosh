//go:build !windows
// +build !windows

package term

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/creack/pty"
	"github.com/tsukinoko-kun/ohmygosh/internal/config"
	"github.com/tsukinoko-kun/ohmygosh/internal/ui/exit"
)

func InheritSize() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGWINCH)
	go func() {
		for range c {
			if size, err := pty.GetsizeFull(os.Stdin); err == nil {
				Cols = size.Cols - 2 // padding
				config.SetEnviron("COLUMNS", strconv.Itoa(int(Cols)))
				config.SetEnviron("LINES", strconv.Itoa(int(size.Rows)))
				_ = exit.InheritSize(size)
			}
		}
	}()
	// send one now to set initial size
	c <- syscall.SIGWINCH
}

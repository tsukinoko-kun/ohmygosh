//go:build !windows
// +build !windows

package term

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"github.com/tsukinoko-kun/ohmygosh/internal/ui/exit"
)

func InheritSize() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGWINCH)
	go func() {
		for range c {
			if size, err := pty.GetsizeFull(os.Stdin); err == nil {
				_ = exit.InheritSize(size)
			}
		}
	}()
	// send one now to set initial size
	c <- syscall.SIGWINCH
}

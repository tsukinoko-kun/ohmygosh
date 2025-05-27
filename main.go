package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	zone "github.com/lrstanley/bubblezone"
	"github.com/tsukinoko-kun/ohmygosh/internal/commands"
	"github.com/tsukinoko-kun/ohmygosh/internal/metadata"
	"github.com/tsukinoko-kun/ohmygosh/internal/shell"
	"github.com/tsukinoko-kun/ohmygosh/internal/term"
	"github.com/tsukinoko-kun/ohmygosh/internal/ui"
	"github.com/tsukinoko-kun/ohmygosh/internal/ui/exit"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(metadata.Version)
		return
	}

	go term.InheritSize()
	go processSignals()
	go shell.Init()
	defer shell.ClearIPC()

	zone.NewGlobal()
	defer zone.Close()

	if err := ui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(exit.TrackedCommands))
	for _, cmd := range exit.TrackedCommands {
		go func() {
			_ = commands.TerminateCommand(cmd.Cmd)
			wg.Done()
		}()
	}
	wg.Wait()

	if exit.ExitCode != 0 {
		zone.Close()
		shell.ClearIPC()
		os.Exit(exit.ExitCode)
	}
}

func processSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	<-c
	exit.ExitCode = 130
	shell.ClearIPC()

	wg := sync.WaitGroup{}
	wg.Add(len(exit.TrackedCommands))
	for _, cmd := range exit.TrackedCommands {
		go func() {
			_ = commands.TerminateCommand(cmd.Cmd)
			wg.Done()
		}()
	}
	wg.Wait()

	zone.Close()
	os.Exit(exit.ExitCode)
}

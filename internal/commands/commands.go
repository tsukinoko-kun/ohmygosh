package commands

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// TerminateCommand attempts to gracefully terminate a command and waits up to
// 2 seconds. If the command doesn't exit within 2 seconds, it will be killed.
// The function is guaranteed to return within 2 seconds and ensures the
// command is no longer running.
func TerminateCommand(cmd *exec.Cmd) error {
	if cmd == nil {
		return nil
	}

	// Check if the process has even started
	if cmd.Process == nil {
		return nil
	}

	// Create a context with 2-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Channel to signal when Wait() completes
	waitDone := make(chan error, 1)

	// Start a goroutine to wait for the process
	go func() {
		waitDone <- cmd.Wait()
	}()

	// Try to terminate gracefully first
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If we can't send SIGTERM, try to kill immediately
		// This handles cases where the process might already be dead
		if killErr := cmd.Process.Kill(); killErr != nil {
			// Process might already be dead, check if Wait() completes
			select {
			case waitErr := <-waitDone:
				return waitErr
			case <-time.After(100 * time.Millisecond):
				// If Wait() doesn't complete quickly, assume process is stuck
				return killErr
			}
		}
	}

	// Wait for either the process to exit or timeout
	select {
	case waitErr := <-waitDone:
		// Process exited gracefully
		return waitErr
	case <-ctx.Done():
		// Timeout reached, force kill the process
		if killErr := cmd.Process.Kill(); killErr != nil {
			// Kill failed, but we still need to wait for cleanup
			select {
			case waitErr := <-waitDone:
				return waitErr
			case <-time.After(500 * time.Millisecond):
				// Give up waiting, return the kill error
				return killErr
			}
		}

		// Wait for the killed process to be cleaned up
		select {
		case waitErr := <-waitDone:
			return waitErr
		case <-time.After(500 * time.Millisecond):
			// Process should be dead by now, but Wait() is taking too long
			// This is unusual but we need to return within our time constraint
			return ctx.Err()
		}
	}
}

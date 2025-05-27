package prompt

import (
	"os/exec"
	"strings"
)

func gitBranch() string {
	output := strings.Builder{}

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branch, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return ""
		}
		return " ?"
	}

	output.WriteString("  ")
	output.WriteString(strings.TrimSpace(string(branch)))

	cmd = exec.Command("git", "status", "--porcelain")
	status, err := cmd.Output()
	if err != nil || len(status) <= 1 {
		return output.String()
	}

	stashed := false
	cmd = exec.Command("git", "stash", "list")
	stash, err := cmd.Output()
	if err == nil && len(stash) > 1 {
		stashed = true
	}
	modified := false
	untracked := false
	staged := false
	lines := strings.Split(string(status), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, " M ") {
			modified = true
		}
		if strings.HasPrefix(line, "?? ") {
			untracked = true
		}
		if strings.HasPrefix(line, "A ") {
			staged = true
		}
	}

	if modified || untracked || staged || stashed {
		output.WriteString(" [")
	}
	if stashed {
		output.WriteString("$")
	}
	if modified {
		output.WriteString("!")
	}
	if staged {
		output.WriteString("+")
	} else if untracked {
		output.WriteString("?")
	}
	if modified || untracked || staged || stashed {
		output.WriteString("]")
	}

	return output.String()
}

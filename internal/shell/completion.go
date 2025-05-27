package shell

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tsukinoko-kun/ohmygosh/internal/config"
)

var maxResults = -1

func getMaxResults() int {
	if maxResults == -1 {
		if termHeight, err := strconv.Atoi(os.Getenv("LINES")); err == nil {
			maxResults = termHeight - 4
		} else {
			maxResults = 20
		}
	}
	return maxResults
}

func init() {
	if termHeight, err := strconv.Atoi(os.Getenv("LINES")); err == nil {
		maxResults = termHeight - 4
	}
}

// Completion represents a shell completion result
type Completion struct {
	Value   string
	Display string
}

// GetCompletions returns completions for the given shell, command, and cursor position
func GetCompletions(
	command string,
	cursorPos int,
) ([]Completion, error) {
	shell := config.Get.Shell.Completion
	switch strings.ToLower(filepath.Base(shell)) {
	case "bash":
		return getBashCompletions(command, cursorPos)
	case "zsh":
		return getZshCompletions(command, cursorPos)
	case "powershell", "pwsh", "powershell.exe", "pwsh.exe":
		return getPowerShellCompletions(shell, command, cursorPos)
	default:
		return nil, fmt.Errorf("unsupported shell: '%s'", shell)
	}
}

// getBashCompletions gets completions from Bash
func getBashCompletions(command string, cursorPos int) ([]Completion, error) {
	if cursorPos > len(command) {
		cursorPos = len(command)
	}

	// Extract the word being completed
	beforeCursor := command[:cursorPos]
	words := strings.Fields(beforeCursor)

	var currentWord string
	if len(words) > 0 && !strings.HasSuffix(beforeCursor, " ") {
		currentWord = words[len(words)-1]
	}

	// Create a bash script that sets up completion and gets results
	script := fmt.Sprintf(`
set -e
export COMP_LINE="%s"
export COMP_POINT=%d
export COMP_WORDS=(%s)
export COMP_CWORD=%d

# Try to get completions using compgen
if [ -n "%s" ]; then
    compgen -f -c -d -- "%s" 2>/dev/null || true
else
    compgen -c 2>/dev/null || true
fi
`,
		strings.ReplaceAll(command, `"`, `\"`),
		cursorPos,
		formatBashArray(strings.Fields(beforeCursor)),
		len(strings.Fields(beforeCursor))-1,
		currentWord,
		currentWord,
	)

	cmd := exec.Command("bash", "-l", "-c", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bash completion failed: %w", err)
	}

	return parseBashOutput(string(output), currentWord), nil
}

// getZshCompletions gets completions from Zsh
func getZshCompletions(command string, cursorPos int) ([]Completion, error) {
	if cursorPos > len(command) {
		cursorPos = len(command)
	}

	beforeCursor := command[:cursorPos]
	words := strings.Fields(beforeCursor)

	var currentWord string
	if len(words) > 0 && !strings.HasSuffix(beforeCursor, " ") {
		currentWord = words[len(words)-1]
	}

	// Use zsh's native completion capabilities
	script := fmt.Sprintf(`
# Set up basic completion
autoload -U compinit
compinit -u 2>/dev/null

# Function to get completions
get_completions() {
    local word="$1"
    local line="$2"
    
    # Get command completions
    if [[ -z "$word" ]] || [[ "$line" =~ '^[[:space:]]*[^[:space:]]+[[:space:]]*$' ]]; then
        # Complete commands
        print -l ${(k)commands}
        print -l ${(k)aliases}
        print -l ${(k)functions}
        print -l ${(k)builtins}
    else
        # Complete files and directories
        setopt NULL_GLOB
        local matches=()
        
        # File completions
        matches+=(${word}*(.N))
        matches+=(${word}*(/N))
        
        # If no matches, try without prefix
        if [[ ${#matches} -eq 0 ]]; then
            matches+=(*${word}*(.N))
            matches+=(*${word}*(/N))
        fi
        
        # Command completions if word looks like a command
        if [[ "$word" =~ '^[a-zA-Z]' ]]; then
            matches+=(${(M)${(k)commands}:#${word}*})
            matches+=(${(M)${(k)aliases}:#${word}*})
            matches+=(${(M)${(k)functions}:#${word}*})
            matches+=(${(M)${(k)builtins}:#${word}*})
        fi
        
        # Remove duplicates and print
        print -l ${(u)matches}
    fi
}

get_completions "%s" "%s" | head -50
`,
		strings.ReplaceAll(currentWord, `"`, `\"`),
		strings.ReplaceAll(command, `"`, `\"`),
	)

	cmd := exec.Command("zsh", "-l", "-c", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("zsh completion failed: %w", err)
	}

	return parseZshOutput(string(output), currentWord), nil
}

// getPowerShellCompletions gets completions from PowerShell
func getPowerShellCompletions(
	shell string,
	command string,
	cursorPos int,
) ([]Completion, error) {
	if cursorPos > len(command) {
		cursorPos = len(command)
	}

	// PowerShell script to get completions
	script := fmt.Sprintf(`
$inputScript = @'
%s
'@

$cursorPosition = %d

try {
    $completions = TabExpansion2 -inputScript $inputScript -cursorColumn $cursorPosition
    
    $results = @()
    if ($completions -and $completions.CompletionMatches) {
        foreach ($completion in $completions.CompletionMatches) {
            $result = @{
                Value = $completion.CompletionText
                Display = if ($completion.ListItemText) { 
                    $completion.ListItemText 
                } else { 
                    $completion.CompletionText 
                }
            }
            $results += $result
        }
    }
    
    $results | ConvertTo-Json -Depth 2
} catch {
    @() | ConvertTo-Json
}
`,
		strings.ReplaceAll(command, "'", "''"),
		cursorPos,
	)

	cmd := exec.Command(shell, "-NoProfile", "-Command", script)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("powershell completion failed: %w", err)
	}

	return parsePowerShellOutput(string(output))
}

// Helper functions

func formatBashArray(words []string) string {
	quoted := make([]string, len(words))
	for i, word := range words {
		quoted[i] = fmt.Sprintf(`"%s"`, strings.ReplaceAll(word, `"`, `\"`))
	}
	return strings.Join(quoted, " ")
}

func parseBashOutput(output, currentWord string) []Completion {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var completions []Completion

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		completion := Completion{
			Value:   strings.TrimPrefix(line, currentWord),
			Display: line,
		}

		// If we have a current word, show what would be completed
		if currentWord != "" && strings.HasPrefix(line, currentWord) {
			completion.Display = line
		}

		completions = append(completions, completion)

		if len(completions) >= getMaxResults() {
			break
		}
	}

	return completions
}

func parseZshOutput(output, currentWord string) []Completion {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var completions []Completion

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		completion := Completion{
			Value:   strings.TrimPrefix(line, currentWord),
			Display: line,
		}

		completions = append(completions, completion)

		if len(completions) >= getMaxResults() {
			break
		}
	}

	return completions
}

func parsePowerShellOutput(output string) ([]Completion, error) {
	output = strings.TrimSpace(output)
	if output == "" || output == "[]" {
		return []Completion{}, nil
	}

	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		// Try parsing as single object
		var single map[string]interface{}
		if err := json.Unmarshal([]byte(output), &single); err != nil {
			return nil, fmt.Errorf("failed to parse PowerShell output: %w", err)
		}
		results = []map[string]interface{}{single}
	}

	var completions []Completion
	for _, result := range results {
		value, ok := result["Value"].(string)
		if !ok {
			continue
		}

		display, ok := result["Display"].(string)
		if !ok {
			display = value
		}

		completions = append(completions, Completion{
			Value:   value,
			Display: display,
		})

		if len(completions) >= getMaxResults() {
			break
		}
	}

	return completions, nil
}

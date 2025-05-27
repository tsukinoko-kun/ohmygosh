package neofetch

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/tsukinoko-kun/ohmygosh/internal/config"
)

var (
	username string
	hostname string
	userLine string
	shell    string
	terminal string

	detailsKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Bold(true)
)

var (
	green = lipgloss.NewStyle().
		Foreground(lipgloss.Color("2"))
	yellow = lipgloss.NewStyle().
		Foreground(lipgloss.Color("3"))
	red = lipgloss.NewStyle().
		Foreground(lipgloss.Color("1"))
	magenta = lipgloss.NewStyle().
		Foreground(lipgloss.Color("5"))
	blue = lipgloss.NewStyle().
		Foreground(lipgloss.Color("4"))
	white = lipgloss.NewStyle().
		Foreground(lipgloss.Color("7"))
	black = lipgloss.NewStyle().
		Foreground(lipgloss.Color("0"))
)

type PrintUpdateMsg struct{}

func genericInit() {
	u, err := user.Current()
	if err != nil {
		username = "Unknown"
	}
	username = u.Username

	h, err := os.Hostname()
	if err != nil {
		hostname = "Unknown"
	}
	hostname = h

	userLine = strings.Repeat("-", len(username)+len(hostname)+1)

	shell = runCommand(config.Get.Shell.Exe, "--version")

	terminal = os.Getenv("TERM")
	if strings.HasPrefix(terminal, "xterm-") {
		terminal = terminal[6:]
	}
}

func detail(key string, value string) string {
	return detailsKeyStyle.Render(key) + ": " + value
}

// runCommand executes a command and returns its trimmed stdout.
func runCommand(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out.String())
}

type data struct {
	key   string
	value string
}

func build(asciiArt string, data []data) string {
	lines := strings.Split(asciiArt, "\n")
	width := 0
	for _, line := range lines {
		newWidth := lipgloss.Width(line)
		if newWidth > width {
			width = newWidth
		}
	}
	width += 2

	colorPalette := strings.Builder{}
	colorPalette.WriteString("\n")
	widthSpacer := strings.Repeat(" ", width)
	colorPalette.WriteString(widthSpacer)
	s := lipgloss.NewStyle()
	for i := 0; i < 8; i++ {
		c := lipgloss.Color(fmt.Sprintf("%d", i))
		colorPalette.WriteString(s.Background(c).Render("   "))
	}
	colorPalette.WriteString("\n")
	colorPalette.WriteString(widthSpacer)
	for i := 8; i < 16; i++ {
		c := lipgloss.Color(fmt.Sprintf("%d", i))
		colorPalette.WriteString(s.Background(c).Render("   "))
	}

	if len(data) > 0 {
		for i := 0; i < len(data)+2; i++ {
			var line string
			if i >= len(lines) {
				line = strings.Repeat(" ", width)
				lines = append(lines, line)
			} else {
				line = lines[i]
				lineLength := lipgloss.Width(line)
				if lineLength < width {
					line = line + strings.Repeat(" ", width-lineLength)
				}
			}
			switch i {
			case 0:
				lines[i] = line + green.Render(username) + "@" + green.Render(hostname)
			case 1:
				lines[i] = line + userLine
			default:
				d := data[i-2]
				lines[i] = line + detail(d.key, d.value)
			}
		}
	}
	lines = append(lines, colorPalette.String())
	return strings.Join(lines, "\n")
}

func firstMatchOr(text string, or string, regex ...*regexp.Regexp) string {
	results := make([]string, 0, len(regex))
	for _, r := range regex {
		if match := r.FindStringSubmatch(text); len(match) > 0 {
			results = append(results, match[1])
		}
	}
	if len(results) > 0 {
		return strings.Join(results, " ")
	}
	return or
}

func atoiOr(text string, or int) int {
	i, err := strconv.Atoi(text)
	if err != nil {
		return or
	}
	return i
}

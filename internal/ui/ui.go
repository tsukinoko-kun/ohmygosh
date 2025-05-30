package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
	zone "github.com/lrstanley/bubblezone"
	"github.com/tsukinoko-kun/ohmygosh/internal/commands"
	"github.com/tsukinoko-kun/ohmygosh/internal/config"
	"github.com/tsukinoko-kun/ohmygosh/internal/history"
	"github.com/tsukinoko-kun/ohmygosh/internal/neofetch"
	"github.com/tsukinoko-kun/ohmygosh/internal/prompt"
	"github.com/tsukinoko-kun/ohmygosh/internal/shell"
	"github.com/tsukinoko-kun/ohmygosh/internal/term"
	"github.com/tsukinoko-kun/ohmygosh/internal/ui/ansicompiler"
	textinput "github.com/tsukinoko-kun/ohmygosh/internal/ui/bubbles/vimtextinput"
	"github.com/tsukinoko-kun/ohmygosh/internal/ui/exit"
)

type CopyStatus uint8

const (
	CopyStatusNone CopyStatus = iota
	CopyStatusSuccess
	CopyStatusFailure
)

// CommandBlock represents a single command execution in a virtual TTY
type CommandBlock struct {
	Output        strings.Builder
	StartTime     time.Time
	EndTime       time.Time
	Command       string
	Prompt        string
	CopyError     string
	ID            int
	ExitCode      int
	PTY           *os.File
	Cmd           *exec.Cmd
	OutputChan    chan string
	mu            sync.Mutex
	CopyStatus    CopyStatus
	IsRunning     bool
	Focused       bool
	UsesAltScreen bool
	InDirectMode  bool
}

// Model represents the application state
type Model struct {
	Input        textinput.Model
	Viewport     viewport.Model
	Cmp          Cmp
	Commands     []*CommandBlock
	FocusedBlock *CommandBlock
	NextID       int
	Width        int
	Height       int
	Scrolling    bool
}

type Cmp struct {
	Completions []shell.Completion
	Error       error
	Cursor      int
	Active      bool
}

// Message types
type CommandOutputMsg struct {
	Output string
	ID     int
}

type CommandFinishedMsg struct {
	ID int
}

type AltScreenDetectedMsg struct {
	ID int
}

type DirectModeFinishedMsg struct {
	ID       int
	ExitCode int
}

func InitialModel() Model {
	input := textinput.New()
	input.SetMode(textinput.ModeInsert)
	input.Focus()
	input.SetWidth(80)

	viewport := viewport.New(80, 20)
	viewport.KeyMap.PageDown.Unbind()
	viewport.KeyMap.PageUp.Unbind()
	viewport.KeyMap.HalfPageUp.Unbind()
	viewport.KeyMap.HalfPageDown.Unbind()
	viewport.KeyMap.Down.Unbind()
	viewport.KeyMap.Up.Unbind()
	viewport.KeyMap.Left.Unbind()
	viewport.KeyMap.Right.Unbind()
	viewport.SetContent("")

	return Model{
		Commands: []*CommandBlock{},
		Input:    input,
		Viewport: viewport,
		NextID:   1,
	}
}

func ExecuteCommand(cmd string, id int) (*CommandBlock, tea.Cmd) {
	block := &CommandBlock{
		ID:         id,
		Command:    cmd,
		Prompt:     prompt.Get(),
		IsRunning:  true,
		ExitCode:   -1,
		StartTime:  time.Now(),
		OutputChan: make(chan string),
	}

	sh, shellArgs := shell.GetShellArgv()
	fullCmd := exec.Command(sh, append(shellArgs, shell.Escape(shell.Wrap(shell.Aliases()+cmd)))...)
	fullCmd.Env = config.Environ

	ptmx, err := pty.Start(fullCmd)
	if err != nil {
		block.Output.WriteString(fmt.Sprintf("Error: %v\n", err))
		block.IsRunning = false
		block.EndTime = time.Now()
		return block, nil
	}
	exit.TrackCommand(fullCmd, ptmx)

	block.PTY = ptmx
	block.Cmd = fullCmd

	readOutput := func() tea.Msg {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				output := string(buf[:n])

				// Check for alt screen sequences
				if detectAltScreen(output) {
					return AltScreenDetectedMsg{ID: block.ID}
				}

				return CommandOutputMsg{
					ID:     block.ID,
					Output: output,
				}
			}
			if err != nil {
				if err != io.EOF {
					return CommandOutputMsg{
						ID:     block.ID,
						Output: fmt.Sprintf("Error reading: %v\n", err),
					}
				}
				break
			}
		}
		return CommandFinishedMsg{ID: block.ID}
	}

	return block, readOutput
}

func ExecuteCommandFullScreen(cmd string, id int) (*CommandBlock, tea.Cmd) {
	block := &CommandBlock{
		ID:            id,
		Command:       cmd,
		Prompt:        prompt.Get(),
		IsRunning:     true,
		ExitCode:      -1,
		StartTime:     time.Now(),
		OutputChan:    nil,
		UsesAltScreen: true,
		InDirectMode:  true,
	}

	sh, shellArgs := shell.GetShellArgv()
	fullCmd := exec.Command(sh, append(shellArgs, shell.Escape(shell.Wrap(shell.Aliases()+cmd)))...)
	exit.TrackCommand(fullCmd, nil)
	fullCmd.Env = config.Environ
	block.Output.WriteString("[Running in full-screen mode...]\n")

	return block, tea.ExecProcess(fullCmd, func(err error) tea.Msg {
		exitCode := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			} else {
				exitCode = 1
			}
		}
		return DirectModeFinishedMsg{ID: id, ExitCode: exitCode}
	})
}

// Common alt screen sequences:
// \x1b[?1049h - Enter alt screen (used by modern terminals)
// \x1b[?47h   - Enter alt screen (older)
// \x1b[2J     - Clear screen (often used by TUI apps)
// \x1b[H      - Home cursor (often combined with clear)
var altScreenSequences = []string{
	"\x1b[?1049h",   // Modern alt screen
	"\x1b[?47h",     // Older alt screen
	"\x1b[2J\x1b[H", // Clear screen + home (common TUI pattern)
}

// detectAltScreen checks if the output contains alt screen escape sequences
func detectAltScreen(output string) bool {
	for _, seq := range altScreenSequences {
		if strings.Contains(output, seq) {
			return true
		}
	}
	return false
}

func executeInDirectMode(id int, cmd string) tea.Cmd {
	sh, shellArgs := shell.GetShellArgv()
	fullCmd := exec.Command(sh, append(shellArgs, shell.Escape(shell.Wrap(shell.Aliases()+cmd)))...)
	exit.TrackCommand(fullCmd, nil)
	fullCmd.Env = config.Environ

	return tea.ExecProcess(fullCmd, func(err error) tea.Msg {
		exitCode := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			} else {
				exitCode = 1
			}
		}
		return DirectModeFinishedMsg{ID: id, ExitCode: exitCode}
	})
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.Cmp.Active {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.Cmp.Active = false
			case "enter":
				m.Cmp.Active = false
				m.Input.InsertText(m.Cmp.Completions[m.Cmp.Cursor].Value)
				m.Cmp.Completions = nil
				return m, nil
			case "tab", "up":
				m.Cmp.Cursor--
				if m.Cmp.Cursor < 0 {
					m.Cmp.Cursor = len(m.Cmp.Completions) - 1
				}
			case "down":
				m.Cmp.Cursor++
				if m.Cmp.Cursor >= len(m.Cmp.Completions) {
					m.Cmp.Cursor = 0
				}
			}
			return m, nil
		}
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case neofetch.PrintUpdateMsg:
		m.updateViewContent()
	case tea.KeyMsg:
		// Handle viewport scrolling keys when not focused on a command
		if m.FocusedBlock == nil {
			switch msg.String() {
			case "pgup", "pgdown", "home", "end":
				m.Scrolling = true
				var cmd tea.Cmd
				m.Viewport, cmd = m.Viewport.Update(msg)
				return m, cmd
			default:
				// Reset scrolling flag for non-navigation keys
				m.Scrolling = false
			}
		}

		// Global keybindings
		switch msg.String() {
		case "enter":
			if cmd := strings.TrimSpace(m.Input.Value()); cmd != "" {
				return enterCommand(m, cmd)
			}

		case "esc":
			// Return focus to command prompt
			if m.FocusedBlock != nil {
				m.FocusedBlock.Focused = false
				m.FocusedBlock = nil
				m.Input.Focus()
				m.updateViewContent()
			} else {
				if m.FocusedBlock == nil {
					var cmd tea.Cmd
					m.Input, cmd = m.Input.Update(msg)
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				}
			}
			return m, nil

		case "tab":
			if m.FocusedBlock == nil && !m.Cmp.Active && m.Input.Focused() {
				cmp, err := shell.GetCompletions(m.Input.Value(), m.Input.Cursor())
				if err == nil {
					m.Cmp.Completions = cmp
					m.Cmp.Error = nil
				} else {
					m.Cmp.Completions = nil
					m.Cmp.Error = err
				}
				m.Cmp.Active = true
				return m, nil
			}

		case "ctrl+up":
			// Cycle focus through running commands
			running := []*CommandBlock{}
			for _, block := range m.Commands {
				if block.IsRunning {
					running = append(running, block)
				}
			}

			// Reverse to prioritize most recent commands
			for i, j := 0, len(running)-1; i < j; i, j = i+1, j-1 {
				running[i], running[j] = running[j], running[i]
			}

			if len(running) > 0 {
				// Clear current focus
				if m.FocusedBlock != nil {
					m.FocusedBlock.Focused = false
				}

				// Find next block to focus
				nextIdx := 0
				if m.FocusedBlock != nil {
					for i, block := range running {
						if block.ID == m.FocusedBlock.ID {
							nextIdx = (i + 1) % len(running)
							break
						}
					}
				}

				m.FocusedBlock = running[nextIdx]
				m.FocusedBlock.Focused = true

				// Auto-scroll to bottom for new output, unless user is manually scrolling
				if !m.Scrolling {
					m.Viewport.GotoBottom()
				}

				// Update the view
				m.updateViewContent()
				return m, nil
			}

		case "ctrl+down":
			// Cycle focus through running commands
			running := []*CommandBlock{}
			for _, block := range m.Commands {
				if block.IsRunning {
					running = append(running, block)
				}
			}

			if len(running) > 0 {
				// Clear current focus
				if m.FocusedBlock != nil {
					m.FocusedBlock.Focused = false
				}

				// Find next block to focus
				nextIdx := 0
				if m.FocusedBlock != nil {
					for i, block := range running {
						if block.ID == m.FocusedBlock.ID {
							nextIdx = (i + 1) % len(running)
							break
						}
					}
				}

				m.FocusedBlock = running[nextIdx]
				m.FocusedBlock.Focused = true

				// Auto-scroll to bottom for new output, unless user is manually scrolling
				if !m.Scrolling {
					m.Viewport.GotoBottom()
				}

				// Update the view
				m.updateViewContent()
				return m, nil
			}
		}

		// If a block is focused, send input to its PTY
		if m.FocusedBlock != nil && m.FocusedBlock.IsRunning {
			if msg.Paste {
				cb, _ := clipboard.ReadAll()
				if _, err := m.FocusedBlock.PTY.WriteString(cb); err != nil {
					m.FocusedBlock.Output.WriteString(fmt.Sprintf("Error sending input: %v\n", err))
					m.updateViewContent()
				}
				return m, nil
			}
			switch msg.String() {
			case "enter":
				if _, err := m.FocusedBlock.PTY.WriteString("\n"); err != nil {
					m.FocusedBlock.Output.WriteString(fmt.Sprintf("Error sending input: %v\n", err))
				}
			case "backspace":
				if _, err := m.FocusedBlock.PTY.WriteString("\b \b"); err != nil {
					m.FocusedBlock.Output.WriteString(fmt.Sprintf("Error sending input: %v\n", err))
				}
			default:
				for _, r := range msg.Runes {
					b := make([]byte, utf8.RuneLen(r))
					utf8.EncodeRune(b, r)
					if _, err := m.FocusedBlock.PTY.Write(b); err != nil {
						m.FocusedBlock.Output.WriteString(fmt.Sprintf("Error sending input: %v\n", err))
					}
				}
			}
			m.updateViewContent()
			return m, nil
		}

		// Otherwise update the input field
		var cmd tea.Cmd
		m.Input, cmd = m.Input.Update(msg)
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		// Update viewport dimensions
		m.Viewport.Width = msg.Width
		m.Viewport.Height = msg.Height - 2 // Leave space for input

		// Update input width
		m.Input.SetWidth(msg.Width)

		m.updateViewContent()

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.Scrolling = true
			var cmd tea.Cmd
			m.Viewport, cmd = m.Viewport.Update(msg)
			return m, cmd

		case tea.MouseButtonWheelDown:
			m.Scrolling = true
			var cmd tea.Cmd
			m.Viewport, cmd = m.Viewport.Update(msg)
			return m, cmd

		case tea.MouseButtonLeft:
			for _, block := range m.Commands {
				if zone.Get(fmt.Sprintf("block_copy_%d", block.ID)).InBounds(msg) {
					err := clipboard.WriteAll(fmt.Sprintf("$ %s\n%s", block.Command, ansicompiler.CompileAnsi(block.Output.String())))
					if err != nil {
						block.CopyStatus = CopyStatusFailure
						block.CopyError = err.Error()
					} else {
						block.CopyStatus = CopyStatusSuccess
					}
					m.updateViewContent()
					return m, nil
				} else if zone.Get(fmt.Sprintf("block_cancel_%d", block.ID)).InBounds(msg) {
					block.IsRunning = false
					block.ExitCode = 130
					block.CopyStatus = CopyStatusNone
					_ = commands.TerminateCommand(block.Cmd)
					m.updateViewContent()
					return m, nil
				}
			}

		default:
			m.Scrolling = true
		}

	case AltScreenDetectedMsg:
		for _, block := range m.Commands {
			if block.ID == msg.ID {
				// Clean up PTY
				if block.PTY != nil {
					_ = block.PTY.Close()
				}
				if block.Cmd != nil && block.Cmd.Process != nil {
					_ = commands.TerminateCommand(block.Cmd)
				}

				block.UsesAltScreen = true
				block.InDirectMode = true
				block.Output.Reset()
				block.Output.WriteString("[Restarting in full-screen mode...]\n")

				m.updateViewContent()

				// Execute in direct mode
				return m, executeInDirectMode(block.ID, block.Command)
			}
		}

	case DirectModeFinishedMsg:
		// Find the block that was in direct mode and mark it as finished
		for _, block := range m.Commands {
			if block.InDirectMode {
				block.InDirectMode = false
				block.IsRunning = false
				block.ExitCode = msg.ExitCode
				block.EndTime = time.Now()

				m.updateViewContent()
				break
			}
		}

	case CommandOutputMsg:
		// Find the command block and update its output
		for _, block := range m.Commands {
			if block.ID == msg.ID {
				block.mu.Lock()
				block.Output.WriteString(msg.Output)
				block.mu.Unlock()
				m.updateViewContent()

				// Continue reading from the PTY
				cmds = append(cmds, func() tea.Msg {
					buf := make([]byte, 4096)
					n, err := block.PTY.Read(buf)
					if n > 0 {
						return CommandOutputMsg{
							ID:     block.ID,
							Output: string(buf[:n]),
						}
					}
					if err != nil {
						if err != io.EOF {
							return CommandOutputMsg{
								ID:     block.ID,
								Output: fmt.Sprintf("Error reading: %v\n", err),
							}
						}
						return CommandFinishedMsg{ID: block.ID}
					}
					return nil
				})
				break
			}
		}

	case CommandFinishedMsg:
		// Mark command as finished
		for _, block := range m.Commands {
			if block.ID == msg.ID {
				block.mu.Lock()
				block.IsRunning = false
				block.EndTime = time.Now()
				if ps, err := block.Cmd.Process.Wait(); err == nil {
					block.ExitCode = ps.ExitCode()
				} else if exitError, ok := err.(*exec.ExitError); ok {
					block.ExitCode = exitError.ExitCode()
				} else {
					block.ExitCode = -1
				}
				block.mu.Unlock()

				// If this was the focused block, clear focus
				if m.FocusedBlock != nil && m.FocusedBlock.ID == block.ID {
					m.FocusedBlock.Focused = false
					m.FocusedBlock = nil
				}

				m.updateViewContent()
				break
			}
		}
	}

	// Handle viewport update
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) updateViewContent() {
	var content strings.Builder

	// Style definitions
	blockStyle := lipgloss.NewStyle().
		MarginBottom(1).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(config.Get.Ui.BorderColor)).
		BorderLeft(true)

	focusedBlockStyle := blockStyle.
		BorderStyle(lipgloss.ThickBorder()).
		BorderRight(false).
		BorderLeft(true).
		BorderTop(false).
		BorderBottom(false).
		BorderForeground(lipgloss.Color(config.Get.Ui.BorderColorFocus))

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.Get.Ui.HeaderColor))

	runningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.Get.Ui.RunningColor))

	completedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.Get.Ui.CompletedColor))

	failedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(config.Get.Ui.FailedColor))

	if neofetch.Print != "" {
		content.WriteString(neofetch.Print + "\n")
	}

	// Render each command block
	for _, block := range m.Commands {
		block.mu.Lock()

		// Choose appropriate styles
		style := blockStyle
		if block.Focused {
			style = focusedBlockStyle
		}

		headerCommandStyle := lipgloss.NewStyle()

		// Create status indicator
		var statusStr string
		if block.IsRunning {
			statusStr = zone.Mark(fmt.Sprintf("block_cancel_%d", block.ID), runningStyle.Render(""))
			headerCommandStyle = headerCommandStyle.Foreground(lipgloss.Color(config.Get.Ui.HeaderCommandColorRunning))
		} else {
			if block.ExitCode != 0 {
				headerCommandStyle = headerCommandStyle.Foreground(lipgloss.Color(config.Get.Ui.HeaderCommandColorFailed))
				statusStr = failedStyle.Render(fmt.Sprintf("✗ %d", block.ExitCode))
			} else {
				headerCommandStyle = headerCommandStyle.Foreground(lipgloss.Color(config.Get.Ui.HeaderCommandColorDone))
				duration := block.EndTime.Sub(block.StartTime)
				if duration > 3*time.Second {
					duration = duration.Round(time.Second)
				} else {
					duration = duration.Round(time.Millisecond)
				}
				statusStr = completedStyle.Render(fmt.Sprintf("✓ (%s)", duration))
			}
		}

		if block.IsRunning {
			headerCommandStyle = headerCommandStyle.Foreground(lipgloss.Color("7"))
		}

		copyButtonContent := " "
		switch block.CopyStatus {
		case CopyStatusSuccess:
			copyButtonContent += "✓"
		case CopyStatusFailure:
			copyButtonContent += "✗ " + block.CopyError
		}
		copyButtonContent = lipgloss.NewStyle().Foreground(lipgloss.Color(config.Get.Ui.HeaderCopyColor)).Render(copyButtonContent)
		copyBtn := zone.Mark(fmt.Sprintf("block_copy_%d", block.ID), copyButtonContent)

		// Format header with command and status
		header := headerStyle.Render(fmt.Sprintf("%s %s\n%s %s", block.Prompt, copyBtn, statusStr, headerCommandStyle.Render(block.Command)))

		var blockContent string
		if block.InDirectMode {
			// Show a placeholder for blocks running in direct mode
			blockContent = header + "\n\n[Running in full-screen mode - press any key to return when finished]"
		} else {
			// Normal rendering
			output := ansicompiler.CompileAnsi(block.Output.String())
			blockContent = header + "\n\n" + output
		}
		// Render the full block
		content.WriteString(style.Render(blockContent) + "\n")

		block.mu.Unlock()
	}

	// Update viewport content
	m.Viewport.SetContent(content.String())

	// Auto-scroll to bottom for new content, unless user is manually scrolling
	if !m.Scrolling {
		m.Viewport.GotoBottom()
		m.Scrolling = false
	}
}

func (m Model) View() string {
	if m.Cmp.Active {
		return m.CmpView()
	}
	hasRunningBlocks := false
	for _, block := range m.Commands {
		if block.IsRunning {
			hasRunningBlocks = true
			break
		}
	}
	var osc string
	if !hasRunningBlocks {
		osc = term.PromptEnd
	}
	return zone.Scan(fmt.Sprintf(
		"%s\n%s",
		m.Viewport.View(),
		lipgloss.NewStyle().
			Render(m.Input.View()),
	)) + osc
}

func Run() error {
	exit.P = tea.NewProgram(
		InitialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := exit.P.Run(); err != nil {
		return err
	}
	return nil
}

var (
	// Style for the modal dialog box
	modalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")). // Purple
			Padding(1, 2).
			Align(lipgloss.Center)

	// Style for focused option in modal
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	// Style for blurred (not focused) option in modal
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	dimmedColor = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#444444"}
)

func (m Model) CmpView() string {
	var renderedOptions []string
	renderedOptions = append(renderedOptions, "(/, Enter to select, Esc to close)")
	for i, opt := range m.Cmp.Completions {
		if i == m.Cmp.Cursor {
			renderedOptions = append(renderedOptions, focusedStyle.Render("[ "+opt.Display+" ]"))
		} else {
			renderedOptions = append(renderedOptions, blurredStyle.Render("  "+opt.Display+"  "))
		}
	}
	modalContent := lipgloss.JoinVertical(lipgloss.Center,
		renderedOptions...,
	)
	modalDialog := modalBoxStyle.Render(modalContent)
	return lipgloss.Place(
		m.Width,
		m.Height,
		lipgloss.Center,
		lipgloss.Center,
		modalDialog,
		lipgloss.WithWhitespaceForeground(dimmedColor),
	)
}

func enterCommand(m Model, cmd string) (Model, tea.Cmd) {
	words := strings.Fields(cmd)
	if len(words) == 0 {
		return m, nil
	}

	m.Input.Reset()
	history.Push(cmd)

	for k, v := range config.Get.Shell.Alias {
		if words[0] == k {
			words[0] = v
			cmd = v + " " + strings.Join(words[1:], " ")
			break
		}
	}

	var (
		block   *CommandBlock
		execCmd tea.Cmd
	)
	if words[0] == "!" {
		block, execCmd = ExecuteCommandFullScreen(cmd[2:], m.NextID)
	}
	for _, app := range fullScreenApps {
		if words[0] == app {
			block, execCmd = ExecuteCommandFullScreen(cmd, m.NextID)
			break
		}
	}
	if block == nil {
		if words[0] == "clear" {
			for _, block := range m.Commands {
				block.mu.Lock()
				_ = commands.TerminateCommand(block.Cmd)
				if block.OutputChan != nil {
					close(block.OutputChan)
				}
				block.Output.Reset()
				_ = block.PTY.Close()
				block.mu.Unlock()
			}
			m.Commands = nil
			exit.ClearTrackedCommands()
			m.NextID = 1
			m.updateViewContent()
			return m, nil
		}
		if len(words) >= 2 {
			fullScreen := false
		complexFullscreenLoop:
			for _, app := range fullScreenAppsComplex {
				if app.Cmd == words[0] {
					for _, subApp := range app.SubCmd {
						if subApp.SubCmd == words[1] {
							if len(subApp.NotArg) > 0 {
								fullScreen = true
								if len(words) >= 3 {
									for _, arg := range subApp.NotArg {
										for _, word := range words[2:] {
											if arg == word {
												fullScreen = false
												break complexFullscreenLoop
											}
										}
									}
								} else {
									break complexFullscreenLoop
								}
							}
							if len(subApp.NotArgStartsWith) > 0 {
								fullScreen = true
								if len(words) >= 3 {
									for _, arg := range subApp.NotArgStartsWith {
										for _, word := range words[2:] {
											if strings.HasPrefix(word, arg) {
												fullScreen = false
												break complexFullscreenLoop
											}
										}
									}
								} else {
									break complexFullscreenLoop
								}
							}
							if len(subApp.Arg) > 0 {
								for _, arg := range subApp.Arg {
									for _, word := range words {
										if arg == word {
											fullScreen = true
											break complexFullscreenLoop
										}
									}
								}
							}
							if !fullScreen && len(subApp.ArgStartsWith) > 0 {
								for _, arg := range subApp.ArgStartsWith {
									for _, word := range words {
										if strings.HasPrefix(word, arg) {
											fullScreen = true
											break complexFullscreenLoop
										}
									}
								}
							}
						}
					}
				}
			}
			if fullScreen {
				block, execCmd = ExecuteCommandFullScreen(cmd, m.NextID)
			}
		}
		if block == nil {
			block, execCmd = ExecuteCommand(cmd, m.NextID)
		}
	}
	m.Commands = append(m.Commands, block)
	m.NextID++

	// Update the view after adding the command
	m.updateViewContent()

	return m, execCmd
}

type (
	FullScreenCmd struct {
		Cmd    string
		SubCmd []FullScreenSubCmd
	}

	FullScreenSubCmd struct {
		SubCmd           string
		NotArg           []string
		Arg              []string
		ArgStartsWith    []string
		NotArgStartsWith []string
	}
)

var fullScreenAppsComplex = []FullScreenCmd{
	{Cmd: "git", SubCmd: []FullScreenSubCmd{
		{SubCmd: "commit",
			NotArg:           []string{"-m", "-F", "-C", "--no-edit"},
			NotArgStartsWith: []string{"--message=", "--file=", "--reuse-message=", "--fixup=", "--squash="},
		},
		{SubCmd: "rebase",
			Arg:           []string{"-i"},
			ArgStartsWith: []string{"--interactive="},
		},
		{SubCmd: "config",
			Arg: []string{"--edit"},
		},
	}},
}

var fullScreenApps = []string{
	"nvim",
	//"docker",
	"lazygit",
	"vim",
	"emacs",
	"nano",
	"zsh",
	"bash",
	"sh",
	"dash",
	"fish",
	"elvish",
	"tmux",
	"htop",
	"btop",
	"pwsh",
	"ssh",
}

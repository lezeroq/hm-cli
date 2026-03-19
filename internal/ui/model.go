// internal/ui/model.go
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hm/internal/claude"
)

type mode int

const (
	modeNormal mode = iota
	modeRefine
	modeWaiting // waiting for Claude during refine
)

// AskFunc is the signature for calling claude (injectable for testing).
type AskFunc func(query string) (*claude.Result, error)

// AskResultMsg is sent back to the model when a refine call completes.
// Exported so tests can inject it directly.
type AskResultMsg struct {
	Command   string
	SessionID string
	Err       error
}

// Model is the Bubble Tea model for the hm TUI.
type Model struct {
	command    string
	mode       mode
	shouldCopy bool
	termWidth  int
	errMsg     string

	input   textinput.Model
	spinner spinner.Model
	askFn   AskFunc
}

var (
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
	hintStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9"))
)

// New creates a Model. askFn may be nil if refine is not used.
func New(command string, askFn AskFunc) Model {
	ti := textinput.New()
	ti.Placeholder = "follow-up prompt..."
	ti.CharLimit = 500

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{
		command:   command,
		mode:      modeNormal,
		termWidth: 80,
		input:     ti,
		spinner:   sp,
		askFn:     askFn,
	}
}

// ShouldCopy reports whether the user pressed Enter to copy to clipboard.
func (m Model) ShouldCopy() bool { return m.shouldCopy }

// Command returns the current command string.
func (m Model) Command() string { return m.command }

// IsRefining reports whether the model is in refine input mode.
func (m Model) IsRefining() bool { return m.mode == modeRefine }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width < m.termWidth {
			// Terminal narrowed: old wide lines visually wrap at the new width,
			// making Bubble Tea's logical-line cursor-up insufficient to erase them.
			// Clear the screen before re-rendering to avoid overlapping boxes.
			m.termWidth = msg.Width
			return m, tea.ClearScreen
		}
		m.termWidth = msg.Width
		return m, nil

	case AskResultMsg:
		m.mode = modeNormal
		if msg.Err != nil {
			m.errMsg = msg.Err.Error()
		} else {
			m.command = msg.Command
			m.errMsg = ""
		}
		return m, nil

	case spinner.TickMsg:
		if m.mode == modeWaiting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeNormal:
		return m.handleNormalKey(msg)
	case modeRefine:
		return m.handleRefineKey(msg)
	case modeWaiting:
		return m, nil // ignore keys while waiting
	}
	return m, nil
}

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := string(msg.Runes)
	switch {
	case msg.Type == tea.KeyEsc, key == "q", key == "n":
		return m, tea.Quit

	case msg.Type == tea.KeyEnter:
		if strings.TrimSpace(m.command) == "" {
			return m, nil
		}
		m.shouldCopy = true
		return m, tea.Quit

	case key == "e":
		m.mode = modeRefine
		m.input.SetValue("")
		m.errMsg = ""
		return m, m.input.Focus()
	}
	return m, nil
}

func (m Model) handleRefineKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		return m, nil

	case tea.KeyEnter:
		query := strings.TrimSpace(m.input.Value())
		if query == "" {
			return m, nil
		}
		m.mode = modeWaiting
		askFn := m.askFn
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg {
				result, err := askFn(query)
				if err != nil {
					return AskResultMsg{Err: err}
				}
				return AskResultMsg{Command: result.Command, SessionID: result.SessionID}
			},
		)

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
}

func (m Model) View() string {
	var sb strings.Builder

	displayCmd := m.command
	if strings.TrimSpace(displayCmd) == "" {
		displayCmd = "No command returned — press [e] to refine or [esc] to dismiss"
	}

	width := m.termWidth - 4
	if width < 20 {
		width = 20
	}
	sb.WriteString(boxStyle.Width(width).Render(displayCmd))
	sb.WriteString("\n")

	if m.errMsg != "" {
		sb.WriteString(errorStyle.Render("error: " + m.errMsg))
		sb.WriteString("\n")
	}

	switch m.mode {
	case modeNormal:
		hints := "[enter] copy   [e] refine   [esc/q/n] dismiss"
		if strings.TrimSpace(m.command) == "" {
			hints = "[e] refine   [esc/q/n] dismiss"
		}
		sb.WriteString(hintStyle.Render(hints))

	case modeRefine:
		sb.WriteString("refine: ")
		sb.WriteString(m.input.View())

	case modeWaiting:
		sb.WriteString(m.spinner.View())
		sb.WriteString(" thinking...")
	}

	return sb.String()
}

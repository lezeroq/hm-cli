// main.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"hm/internal/claude"
	"hm/internal/clipboard"
	"hm/internal/config"
	"hm/internal/ui"
)

// version is injected at build time: go build -ldflags="-X main.version=1.0.0"
var version = "dev"

const usage = `Usage: hm "<query>"

Options:
  --refresh          Clear the Claude session (optionally followed by a query)
  --no-session       Run without session persistence
  --version          Print version and exit
  --help, -h         Show this help

Examples:
  hm "get pods from all namespaces sorted by creation time"
  hm --refresh "list kubernetes contexts"
  hm --no-session "find files larger than 100MB"
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "hm: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	var doRefresh, noSession bool
	var queryParts []string

	for _, a := range args {
		switch a {
		case "--refresh":
			doRefresh = true
		case "--no-session":
			noSession = true
		case "--version":
			fmt.Printf("hm v%s\n", version)
			return nil
		case "--help", "-h":
			fmt.Fprint(os.Stderr, usage)
			return nil
		default:
			queryParts = append(queryParts, a)
		}
	}

	query := strings.Join(queryParts, " ")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Handle --refresh
	if doRefresh {
		if err := cfg.ClearSessionID(); err != nil {
			fmt.Fprintf(os.Stderr, "hm: warning: could not clear session ID: %v\n", err)
		}
		if query == "" {
			fmt.Fprintln(os.Stderr, "Session cleared.")
			return nil
		}
	}

	// Verify claude CLI is available
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found. Install from https://claude.ai/code")
	}

	sessionID := ""
	if !noSession {
		sessionID = cfg.SessionID
	}

	// Pre-TUI progress indicator on stderr
	fmt.Fprint(os.Stderr, "\rThinking...")

	result, callErr := claude.New(cfg.SystemPrompt, sessionID, nil).Ask(query)
	// Clear the spinner line
	fmt.Fprint(os.Stderr, "\r\033[K")

	if callErr != nil {
		if sessionID != "" {
			// Stale session: clear and retry once without session ID
			if err := cfg.ClearSessionID(); err != nil {
				fmt.Fprintf(os.Stderr, "hm: warning: could not clear session ID: %v\n", err)
			}
			result, callErr = claude.New(cfg.SystemPrompt, "", nil).Ask(query)
		}
		if callErr != nil {
			return callErr
		}
	}

	// Persist the new session ID
	if !noSession && result.SessionID != "" {
		if err := cfg.SaveSessionID(result.SessionID); err != nil {
			fmt.Fprintf(os.Stderr, "hm: warning: could not save session ID: %v\n", err)
		}
	}

	// activeSessionID is updated by the askFn closure on each refine call
	activeSessionID := result.SessionID
	if noSession {
		activeSessionID = ""
	}

	askFn := func(followUp string) (*claude.Result, error) {
		r, err := claude.New(cfg.SystemPrompt, activeSessionID, nil).Ask(followUp)
		if err == nil && !noSession && r.SessionID != "" {
			activeSessionID = r.SessionID
			if saveErr := cfg.SaveSessionID(r.SessionID); saveErr != nil {
				fmt.Fprintf(os.Stderr, "hm: warning: could not save session ID: %v\n", saveErr)
			}
		}
		return r, err
	}

	// Launch Bubble Tea TUI
	m := ui.New(result.Command, askFn)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	final := finalModel.(ui.Model)
	cmd := final.Command()

	// Copy to clipboard if user pressed Enter
	if final.ShouldCopy() && strings.TrimSpace(cmd) != "" {
		if err := clipboard.Copy(cfg.ClipboardCmd, cmd); err != nil {
			fmt.Fprintf(os.Stderr, "(clipboard unavailable — command printed above)\n")
		}
	}

	return nil
}

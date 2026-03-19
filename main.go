// main.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
  --continue, -c     Continue from the last command with feedback
  --quiet, -q        Copy command to clipboard and print to stdout, no TUI
  --refresh          Clear the Claude session (optionally followed by a query)
  --no-session       Run without session persistence
  --version          Print version and exit
  --help, -h         Show this help

Examples:
  hm "get pods from all namespaces sorted by creation time"
  hm -c "that showed pods with GPU, I only want CPU pods"
  hm -q "list running containers"
  hm --refresh "list kubernetes contexts"
  hm --no-session "find files larger than 100MB"
`

func main() {
	args := os.Args[1:]
	// When invoked as "hmq", behave as "hm -q"
	if filepath.Base(os.Args[0]) == "hmq" {
		args = append([]string{"-q"}, args...)
	}
	if err := run(args); err != nil {
		fmt.Fprintf(os.Stderr, "hm: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var doRefresh, noSession, quiet bool
	var queryParts []string
	var continueMsg string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--refresh":
			doRefresh = true
		case "--no-session":
			noSession = true
		case "--quiet", "-q":
			quiet = true
		case "--version":
			fmt.Printf("hm v%s\n", version)
			return nil
		case "--help", "-h":
			fmt.Fprint(os.Stderr, usage)
			return nil
		case "--continue", "-c":
			if i+1 < len(args) {
				i++
				continueMsg = args[i]
			}
		default:
			queryParts = append(queryParts, args[i])
		}
	}

	query := strings.Join(queryParts, " ")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Handle --continue: build query from feedback + last known command for context
	if continueMsg != "" {
		if cfg.LastCommand != "" {
			query = fmt.Sprintf("The last command you generated was:\n%s\n\n%s", cfg.LastCommand, continueMsg)
		} else {
			query = continueMsg
		}
	}

	// Handle --refresh before the empty-query guard: "hm --refresh" alone is valid.
	if doRefresh {
		if err := cfg.ClearSessionID(); err != nil {
			fmt.Fprintf(os.Stderr, "hm: warning: could not clear session ID: %v\n", err)
		}
		if query == "" {
			fmt.Fprintln(os.Stderr, "Session cleared.")
			return nil
		}
	}

	if query == "" {
		fmt.Fprint(os.Stderr, usage)
		return fmt.Errorf("no query provided")
	}

	// Verify claude CLI is available
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found. Install from https://claude.ai/code")
	}

	var sessionID string
	if !noSession {
		sessionID = cfg.SessionID
	}

	// Progress indicator (suppressed in quiet mode)
	if !quiet {
		fmt.Fprintln(os.Stderr, "Thinking...")
	}

	result, callErr := claude.New(cfg.SystemPrompt, sessionID, nil).Ask(query)

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

	// Persist the generated command for use with --continue
	if result.Command != "" {
		if err := cfg.SaveLastCommand(result.Command); err != nil {
			fmt.Fprintf(os.Stderr, "hm: warning: could not save last command: %v\n", err)
		}
	}

	if quiet {
		// Quiet mode: copy to clipboard and print, no TUI.
		cmd := result.Command
		if strings.TrimSpace(cmd) != "" {
			if err := clipboard.Copy(cfg.ClipboardCmd, cmd); err != nil {
				fmt.Fprintf(os.Stderr, "hm: warning: clipboard unavailable: %v\n", err)
			}
		}
		fmt.Println(cmd)
		return nil
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

	final, ok := finalModel.(ui.Model)
	if !ok {
		return fmt.Errorf("unexpected model type from TUI: %T", finalModel)
	}
	cmd := final.Command()

	// Copy to clipboard if user pressed Enter
	if final.ShouldCopy() && strings.TrimSpace(cmd) != "" {
		if err := clipboard.Copy(cfg.ClipboardCmd, cmd); err != nil {
			fmt.Fprintf(os.Stderr, "(clipboard unavailable — command printed above)\n")
		}
	}

	return nil
}

// internal/claude/client.go
package claude

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ExecFunc is injectable for testing. Returns stdout, stderr, exit code.
type ExecFunc func(name string, args []string) (stdout, stderr []byte, exitCode int)

// Result holds the parsed response from the claude CLI.
type Result struct {
	Command   string
	SessionID string
}

// Client wraps the claude CLI subprocess.
type Client struct {
	systemPrompt string
	sessionID    string
	exec         ExecFunc
}

type claudeResponse struct {
	Result    string `json:"result"`
	SessionID string `json:"session_id"`
	IsError   bool   `json:"is_error"`
}

// New creates a Client. sessionID may be empty. execFn may be nil (uses os/exec).
func New(systemPrompt, sessionID string, execFn ExecFunc) *Client {
	if execFn == nil {
		execFn = defaultExec
	}
	return &Client{
		systemPrompt: systemPrompt,
		sessionID:    sessionID,
		exec:         execFn,
	}
}

// Ask sends query to the claude CLI and returns the shell command result.
func (c *Client) Ask(query string) (*Result, error) {
	args := []string{
		"-p", query,
		"--output-format", "json",
		"--system-prompt", c.systemPrompt,
	}
	if c.sessionID != "" {
		args = append(args, "--session-id", c.sessionID)
	}

	stdout, stderr, exitCode := c.exec("claude", args)
	if exitCode != 0 {
		// claude writes diagnostics to stderr; fall back to stdout if stderr is empty.
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			msg = strings.TrimSpace(string(stdout))
		}
		return nil, fmt.Errorf("claude exited with code %d: %s", exitCode, msg)
	}

	var resp claudeResponse
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse claude output: %w\nraw: %s", err, stdout)
	}
	if resp.IsError {
		return nil, fmt.Errorf("claude error: %s", resp.Result)
	}

	return &Result{
		Command:   strings.TrimSpace(resp.Result),
		SessionID: resp.SessionID,
	}, nil
}

func defaultExec(name string, args []string) ([]byte, []byte, int) {
	cmd := exec.Command(name, args...)
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return stdout, exitErr.Stderr, exitErr.ExitCode()
		}
		return nil, nil, 1
	}
	return stdout, nil, 0
}

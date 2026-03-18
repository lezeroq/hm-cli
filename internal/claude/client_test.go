// internal/claude/client_test.go
package claude_test

import (
	"encoding/json"
	"testing"

	"hm/internal/claude"
)

func makeExec(stdout string, exitCode int) claude.ExecFunc {
	return func(name string, args []string) ([]byte, []byte, int) {
		return []byte(stdout), nil, exitCode
	}
}

func jsonResponse(result, sessionID string, isError bool) string {
	b, _ := json.Marshal(map[string]interface{}{
		"result":     result,
		"session_id": sessionID,
		"is_error":   isError,
	})
	return string(b)
}

func TestAsk_ReturnsCommandAndSessionID(t *testing.T) {
	exec := makeExec(jsonResponse("kubectl get pods -A", "sess-1", false), 0)
	c := claude.New("system prompt", "", exec)

	result, err := c.Ask("get pods")
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if result.Command != "kubectl get pods -A" {
		t.Errorf("Command = %q, want kubectl get pods -A", result.Command)
	}
	if result.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want sess-1", result.SessionID)
	}
}

func TestAsk_WithSessionID_PassesFlag(t *testing.T) {
	var capturedArgs []string
	execFn := func(name string, args []string) ([]byte, []byte, int) {
		capturedArgs = args
		return []byte(jsonResponse("ls -la", "sess-2", false)), nil, 0
	}
	c := claude.New("prompt", "existing-session", execFn)
	c.Ask("list files")

	found := false
	for i, a := range capturedArgs {
		if a == "--session-id" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "existing-session" {
			found = true
		}
	}
	if !found {
		t.Errorf("--session-id not found in args: %v", capturedArgs)
	}
}

func TestAsk_NoSessionID_OmitsFlag(t *testing.T) {
	var capturedArgs []string
	execFn := func(name string, args []string) ([]byte, []byte, int) {
		capturedArgs = args
		return []byte(jsonResponse("ls", "sess-3", false)), nil, 0
	}
	c := claude.New("prompt", "", execFn)
	c.Ask("list")

	for _, a := range capturedArgs {
		if a == "--session-id" {
			t.Errorf("--session-id should not be passed when sessionID is empty, args: %v", capturedArgs)
		}
	}
}

func TestAsk_IsErrorTrue_ReturnsError(t *testing.T) {
	exec := makeExec(jsonResponse("something went wrong", "", true), 0)
	c := claude.New("prompt", "", exec)

	_, err := c.Ask("get pods")
	if err == nil {
		t.Fatal("expected error when is_error is true")
	}
}

func TestAsk_NonZeroExit_ReturnsError(t *testing.T) {
	exec := makeExec("", 1)
	c := claude.New("prompt", "", exec)

	_, err := c.Ask("get pods")
	if err == nil {
		t.Fatal("expected error on non-zero exit")
	}
}

func TestAsk_EmptyResult_ReturnsEmptyCommand(t *testing.T) {
	exec := makeExec(jsonResponse("", "sess-4", false), 0)
	c := claude.New("prompt", "", exec)

	result, err := c.Ask("do something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Command != "" {
		t.Errorf("expected empty command, got %q", result.Command)
	}
}

func TestAsk_PassesSystemPromptFlag(t *testing.T) {
	var capturedArgs []string
	execFn := func(name string, args []string) ([]byte, []byte, int) {
		capturedArgs = args
		return []byte(jsonResponse("echo hi", "s", false)), nil, 0
	}
	c := claude.New("my system prompt", "", execFn)
	c.Ask("say hi")

	found := false
	for i, a := range capturedArgs {
		if a == "--system-prompt" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "my system prompt" {
			found = true
		}
	}
	if !found {
		t.Errorf("--system-prompt not passed correctly in args: %v", capturedArgs)
	}
}

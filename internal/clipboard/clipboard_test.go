// internal/clipboard/clipboard_test.go
package clipboard_test

import (
	"testing"

	"hm/internal/clipboard"
)

func TestCopy_SuccessfulCommand(t *testing.T) {
	// cat > /dev/null consumes stdin and exits 0
	if err := clipboard.Copy("cat > /dev/null", "test content"); err != nil {
		t.Errorf("Copy() unexpected error: %v", err)
	}
}

func TestCopy_FailingCommand_ReturnsError(t *testing.T) {
	if err := clipboard.Copy("false", "content"); err == nil {
		t.Error("Copy() expected error for failing command, got nil")
	}
}

func TestCopy_InvalidCommand_ReturnsError(t *testing.T) {
	if err := clipboard.Copy("nonexistent-command-xyz", "content"); err == nil {
		t.Error("Copy() expected error for unknown command, got nil")
	}
}

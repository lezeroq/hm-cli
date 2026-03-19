// internal/clipboard/clipboard_test.go
package clipboard_test

import (
	"testing"

	"hm/internal/clipboard"
)

func TestCopy_SuccessfulCommand_NoError(t *testing.T) {
	if err := clipboard.Copy("cat > /dev/null", "test content"); err != nil {
		t.Errorf("Copy() unexpected error: %v", err)
	}
}

// Fire-and-forget: runtime failures (non-zero exit) are not detected.
// The process runs in the background; only start failures are reported.
func TestCopy_FailingCommand_NoError(t *testing.T) {
	// "false" starts fine via sh -c, exits non-zero, but we don't wait
	if err := clipboard.Copy("false", "content"); err != nil {
		t.Errorf("Copy() unexpected error for fire-and-forget command: %v", err)
	}
}

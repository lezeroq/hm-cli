// internal/clipboard/clipboard.go
package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// Copy runs cmd via "sh -c <cmd>" with text piped to stdin.
// The process is started and left running in the background — clipboard tools
// like xclip stay alive until another app pastes, so we must not wait for exit.
// Returns an error only if the command fails to start.
func Copy(cmd, text string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Stdin = strings.NewReader(text)
	if err := c.Start(); err != nil {
		return fmt.Errorf("clipboard command failed to start: %w", err)
	}
	go c.Wait() // reap the process when it eventually exits
	return nil
}

// internal/clipboard/clipboard.go
package clipboard

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Copy runs cmd via "sh -c <cmd>" with text piped to stdin.
// Uses an explicit stdin pipe to guarantee the text is written before returning.
// The process is left running in the background — clipboard tools like xclip
// act as clipboard servers and stay alive until another app pastes.
func Copy(cmd, text string) error {
	c := exec.Command("sh", "-c", cmd)
	stdin, err := c.StdinPipe()
	if err != nil {
		return fmt.Errorf("clipboard stdin pipe: %w", err)
	}
	if err := c.Start(); err != nil {
		return fmt.Errorf("clipboard command failed to start: %w", err)
	}
	// Write text and close pipe synchronously so xclip receives it before we return.
	if _, err := io.WriteString(stdin, text); err != nil {
		fmt.Fprintf(os.Stderr, "hm: warning: clipboard write failed: %v\n", err)
	}
	stdin.Close()
	go c.Wait() // reap the process when it eventually exits
	return nil
}

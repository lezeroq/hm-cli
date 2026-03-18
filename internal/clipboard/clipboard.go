// internal/clipboard/clipboard.go
package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// Copy runs cmd via "sh -c <cmd>" with text piped to stdin.
// Returns an error if the command exits non-zero or cannot be started.
func Copy(cmd, text string) error {
	c := exec.Command("sh", "-c", cmd)
	c.Stdin = strings.NewReader(text)
	if out, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

package claudecode

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// client executes the claude CLI in print mode.
type client struct {
	env []string // Cached environment variables (SSH_AUTH_SOCK filtered out)
}

// newClient creates a new Claude Code CLI client.
func newClient() *client {
	// Filter out SSH_AUTH_SOCK to prevent SSH agent prompts (e.g. 1Password)
	// when claude CLI accesses git repository info.
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "SSH_AUTH_SOCK=") {
			env = append(env, e)
		}
	}
	return &client{env: env}
}

// run executes claude CLI in print mode and returns the output.
func (c *client) run(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	args := []string{
		"-p",
		"--model", model,
		"--system-prompt", systemPrompt,
		"--output-format", "text",
		"--no-session-persistence",
		"--max-turns", "1",
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Env = c.env
	cmd.Stdin = strings.NewReader(userPrompt)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("claude CLI failed: %w: %s", err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude CLI failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

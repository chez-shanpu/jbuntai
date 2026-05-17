package codex

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chez-shanpu/jbuntai/internal/llm/prompt"
)

// finisherImpl implements the Finisher interface using the ChatGPT Responses API.
type finisherImpl struct {
	client          *client
	model           string
	reasoningEffort string
}

// Finish refines rule-based conversion using the ChatGPT Responses API.
func (f *finisherImpl) Finish(ctx context.Context, original, transformed string) (string, error) {
	slog.Default().Debug("codex finisher", "model", f.model, "reasoning_effort", f.reasoningEffort)
	userPrompt := prompt.FormatFinishInput(original, transformed)

	result, err := f.client.run(ctx, f.model, f.reasoningEffort, prompt.FinishSystemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("finish failed: %w", err)
	}

	return result, nil
}

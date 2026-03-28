package codex

import (
	"context"
	"fmt"

	"github.com/chez-shanpu/jbuntai/internal/llm/prompt"
)

// finisherImpl implements the Finisher interface using the ChatGPT Responses API.
type finisherImpl struct {
	client *client
	model  string
}

// Finish refines rule-based conversion using the ChatGPT Responses API.
func (f *finisherImpl) Finish(ctx context.Context, original, transformed string) (string, error) {
	userPrompt := prompt.FormatFinishInput(original, transformed)

	result, err := f.client.run(ctx, f.model, prompt.FinishSystemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("finish failed: %w", err)
	}

	return result, nil
}

package codex

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chez-shanpu/jbuntai/internal/llm"
	"github.com/chez-shanpu/jbuntai/internal/llm/prompt"
)

// disambiguatorImpl implements the Disambiguator interface using the ChatGPT Responses API.
type disambiguatorImpl struct {
	client *client
	model  string
}

// Disambiguate resolves ambiguous particles using the ChatGPT Responses API.
func (d *disambiguatorImpl) Disambiguate(ctx context.Context, items []llm.AmbiguousItem) ([]llm.DisambiguationResult, error) {
	if len(items) == 0 {
		return nil, nil
	}

	userPrompt, err := prompt.FormatDisambiguateInput(items)
	if err != nil {
		return nil, err
	}

	responseText, err := d.client.run(ctx, d.model, prompt.DisambiguateSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("disambiguate failed: %w", err)
	}

	var results []llm.DisambiguationResult
	if err := json.Unmarshal([]byte(prompt.ExtractJSON(responseText)), &results); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return results, nil
}

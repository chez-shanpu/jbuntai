package codex

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"

	"github.com/chez-shanpu/jbuntai/internal/llm"
	"github.com/chez-shanpu/jbuntai/internal/llm/prompt"
)

// disambiguatorImpl implements the Disambiguator interface using Codex app-server.
type disambiguatorImpl struct {
	client          *client
	model           string
	reasoningEffort string
}

// Disambiguate resolves ambiguous particles using Codex app-server.
func (d *disambiguatorImpl) Disambiguate(ctx context.Context, items []llm.AmbiguousItem) ([]llm.DisambiguationResult, error) {
	if len(items) == 0 {
		return nil, nil
	}
	slog.Default().Debug("codex disambiguator", "model", d.model, "reasoning_effort", d.reasoningEffort)

	userPrompt, err := prompt.FormatDisambiguateInput(items)
	if err != nil {
		return nil, err
	}

	responseText, err := d.client.run(ctx, d.model, d.reasoningEffort, prompt.DisambiguateSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("disambiguate failed: %w", err)
	}

	return parseResults(responseText)
}

// parseResults decodes the JSON array of results in the model's response.
func parseResults(responseText string) ([]llm.DisambiguationResult, error) {
	var results []llm.DisambiguationResult
	// Match field names case-insensitively, as encoding/json (v1) did.
	if err := json.Unmarshal([]byte(prompt.ExtractJSON(responseText)), &results, json.MatchCaseInsensitiveNames(true)); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return results, nil
}

package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chez-shanpu/jbuntai/internal/config"
	"github.com/chez-shanpu/jbuntai/internal/llm"
)

// DisambiguatorImpl implements the Disambiguator interface using Claude Code CLI.
type DisambiguatorImpl struct {
	client *Client
	cfg    *config.Config
}

// NewDisambiguator creates a new Claude Code CLI-based disambiguator.
func NewDisambiguator(cfg *config.Config) *DisambiguatorImpl {
	client := NewClient(cfg)
	return &DisambiguatorImpl{client: client, cfg: cfg}
}

const disambiguateSystemPrompt = `You are a Japanese language analysis assistant.
Given a list of ambiguous Japanese particles in context, classify each one according to its grammatical function.
Respond with a JSON array of objects, each with "id" and "answer" fields.
The "answer" must be one of the provided choices for each item.`

// Disambiguate resolves ambiguous particles using Claude Code CLI.
func (d *DisambiguatorImpl) Disambiguate(ctx context.Context, items []llm.AmbiguousItem) ([]llm.DisambiguationResult, error) {
	if len(items) == 0 {
		return nil, nil
	}

	inputJSON, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal items: %w", err)
	}

	userPrompt := fmt.Sprintf("Classify the following ambiguous particles:\n%s", string(inputJSON))

	responseText, err := d.client.run(ctx, d.cfg.LLM.DisambiguateModel, disambiguateSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("disambiguate failed: %w", err)
	}

	var results []llm.DisambiguationResult
	if err := json.Unmarshal([]byte(extractJSON(responseText)), &results); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return results, nil
}

// extractJSON strips markdown code fences and surrounding text from the response,
// then extracts the JSON array.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Handle markdown code fences
	if strings.HasPrefix(s, "```") {
		// Remove first line (```json or ```)
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		// Remove trailing ```
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	// Find JSON array in text (handles surrounding non-JSON text)
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		s = s[start : end+1]
	}
	return s
}

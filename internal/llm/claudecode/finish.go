package claudecode

import (
	"context"
	"fmt"

	"github.com/chez-shanpu/jbuntai/internal/config"
)

// FinisherImpl implements the Finisher interface using Claude Code CLI.
type FinisherImpl struct {
	client *Client
	cfg    *config.Config
}

// NewFinisher creates a new Claude Code CLI-based finisher.
func NewFinisher(cfg *config.Config) *FinisherImpl {
	client := NewClient(cfg)
	return &FinisherImpl{client: client, cfg: cfg}
}

const finishSystemPrompt = `You are a Japanese text editor specializing in "情報文体" (information style).

Information style rules:
- Remove sentence-ending forms (です/ます/だ/である)
- Replace particles with symbols: > (direction), @ (location), → (result), ∵ (reason), , (sequence), : (quotation), 〜 (range), ・ (parallel)
- Delete unnecessary particles: は, が (subject), を (before sino-japanese verbs)
- Use concise, telegram-like expressions
- Preserve technical accuracy and meaning
- Keep code blocks and technical terms unchanged

Given the original Japanese text and a rule-based conversion, refine the conversion to produce natural, readable information style text.
Output ONLY the refined text, no explanations.`

// Finish refines rule-based conversion using Claude Code CLI.
func (f *FinisherImpl) Finish(ctx context.Context, original, transformed string) (string, error) {
	userPrompt := fmt.Sprintf("Original:\n%s\n\nRule-based conversion:\n%s\n\nPlease refine the conversion to natural information style.", original, transformed)

	result, err := f.client.run(ctx, f.cfg.LLM.FinishModel, finishSystemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("finish failed: %w", err)
	}

	if result == "" {
		return "", fmt.Errorf("no text response from CLI")
	}

	return result, nil
}

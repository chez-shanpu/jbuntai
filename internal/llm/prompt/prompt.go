// Package prompt provides shared prompts and utilities for LLM backends.
package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chez-shanpu/jbuntai/internal/llm"
)

// DisambiguateSystemPrompt is the system prompt for particle disambiguation.
const DisambiguateSystemPrompt = `You are a Japanese language analysis assistant.
Given a list of ambiguous Japanese particles in context, classify each one according to its grammatical function.
Respond with a JSON array of objects, each with "id" and "answer" fields.
The "answer" must be one of the provided choices for each item.`

// FinishSystemPrompt is the system prompt for text finishing.
const FinishSystemPrompt = `You are a Japanese text editor specializing in "情報文体" (information style).

Information style rules:
- Remove sentence-ending forms (です/ます/だ/である)
- Replace particles with symbols: > (direction), @ (location), → (result), ∵ (reason), , (sequence), : (quotation), 〜 (range), ・ (parallel)
- Delete unnecessary particles: は, が (subject), を (before sino-japanese verbs)
- Use concise, telegram-like expressions
- Preserve technical accuracy and meaning
- Keep code blocks and technical terms unchanged

Given the original Japanese text and a rule-based conversion, refine the conversion to produce natural, readable information style text.
Output ONLY the refined text, no explanations.`

// FormatDisambiguateInput formats the user prompt for disambiguation.
func FormatDisambiguateInput(items []llm.AmbiguousItem) (string, error) {
	inputJSON, err := json.Marshal(items)
	if err != nil {
		return "", fmt.Errorf("failed to marshal items: %w", err)
	}
	return fmt.Sprintf("Classify the following ambiguous particles:\n%s", string(inputJSON)), nil
}

// FormatFinishInput formats the user prompt for finishing.
func FormatFinishInput(original, transformed string) string {
	return fmt.Sprintf("Original:\n%s\n\nRule-based conversion:\n%s\n\nPlease refine the conversion to natural information style.", original, transformed)
}

// ExtractJSON strips markdown code fences and surrounding text from the response,
// then extracts the JSON array.
func ExtractJSON(s string) string {
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

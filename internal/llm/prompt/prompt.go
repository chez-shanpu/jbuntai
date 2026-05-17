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

Compress as aggressively as possible while preserving all factual information. The rule-based pass is intentionally conservative; you must shorten further wherever lossless compression is possible. Stop only when further removal would lose information. Achievable compression varies by sentence — maximize reduction for each sentence independently.

Symbol set (apply or reinforce):
- > direction/target (に/へ): place > before the target noun, then a space before the verb: "東京に移動" → ">東京 移動"
- @ location (で): place @ before the location noun, then a space before the verb: "会議室で議論" → "@会議室 議論"
- → cause→result chain: use only for clear causation/result, not as a generic particle replacement
- ∵ reason (ので/ため/から): place ∵ before the reason clause, then a space before the result: "予算不足のため延期" → "∵予算不足 延期"
- , sequence (て/で connecting verbs): "調べて報告する" → "調べ,報告"
- : quotation (と言う/と思う): ":X"
- 〜 range (から…まで): "1月〜3月"
- ・ parallel (と/や between nouns): "A・B・C"

Deletion rules (apply or reinforce):
- Drop sentence-ending forms: です/ます/だ/である/ました/ています
- Drop subject particles: は, が (unless adversative が before 、)
- Drop object particle: を before sino-japanese verbs
- Drop sentence-final particles: よ, ね, な, さ
- Drop non-independent nouns: こと, もの

Aggressive compression rules (NOT applied by the rule-based pass — apply them now):
- Delete 連体 "の/への" when the compound noun is unambiguous without it; preserve "の" only when it disambiguates adjacent kanji runs
- Drop "〜について" / "〜に関して" / "〜における" entirely; juxtapose topic with object
- Drop formal noun "という": "〜という[noun]" → "[noun]"
- Drop filler adverbs and demonstratives: さらに, また, この, その, それぞれ, 〜的にも, 〜なども
- Avoid redundant copula or reporting expressions at sentence ends; omit or vary when meaning is clear from context
- Prefer noun-ending sentences; drop trailing copulas
- Keep unmodified: headings (## ...), code blocks, URLs, numbers, proper nouns, technical terms

Output ONLY the refined text. No preamble, no explanation, no code fences.`

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
	return fmt.Sprintf(
		"Original Japanese:\n%s\n\nRule-based conversion:\n%s",
		original, transformed,
	)
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

package tokenizer

import (
	"strings"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"

	"github.com/chez-shanpu/jbuntai/internal/pass"
	"github.com/chez-shanpu/jbuntai/internal/preprocess"
)

// Tokenizer wraps kagome tokenizer.
type Tokenizer struct {
	t *tokenizer.Tokenizer
}

// New creates a new Tokenizer with IPA dictionary.
func New() (*Tokenizer, error) {
	t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		return nil, err
	}
	return &Tokenizer{t: t}, nil
}

// Tokenize converts input text into a slice of Token.
func (tk *Tokenizer) Tokenize(text string) []pass.Token {
	tokens := tk.t.Tokenize(text)
	result := make([]pass.Token, 0, len(tokens))

	for _, t := range tokens {
		features := t.Features()
		pos := extractPOS(features)
		baseForm := pickupFeature(t.BaseForm())
		reading := pickupFeature(t.Reading())

		result = append(result, pass.NewToken(t.Surface, pos, baseForm, reading))
	}

	return result
}

// extractPOS extracts the first 4 features as POS tags.
// In the IPA dictionary, the first 4 features represent the hierarchical POS classification:
//
//	[0] POS (e.g. 名詞, 動詞)
//	[1] POS sub-category 1 (e.g. 固有名詞, 自立)
//	[2] POS sub-category 2 (e.g. 地域, 一般)
//	[3] POS sub-category 3
func extractPOS(features []string) []string {
	pos := make([]string, 0, 4)
	for i := 0; i < 4 && i < len(features); i++ {
		pos = append(pos, features[i])
	}
	return pos
}

// pickupFeature converts kagome's (string, bool) feature result to a clean string.
// Returns empty string for unknown ("*") or missing features.
func pickupFeature(s string, ok bool) string {
	if !ok || s == "*" {
		return ""
	}
	return s
}

// TokenizeSentences splits text into sentences and tokenizes each.
func (tk *Tokenizer) TokenizeSentences(text string) [][]pass.Token {
	var result [][]pass.Token

	lines := strings.SplitSeq(text, "\n")
	for line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(trimmed, preprocess.PlaceholderPrefix) {
			result = append(result, nil)
			continue
		}
		tokens := tk.Tokenize(line)
		result = append(result, tokens)
	}

	return result
}

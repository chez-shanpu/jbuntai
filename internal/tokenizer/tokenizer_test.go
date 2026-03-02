package tokenizer

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tk, err := New()
	if err != nil {
		t.Fatalf("failed to create tokenizer: %v", err)
	}

	tokens := tk.Tokenize("分析します。")

	if len(tokens) == 0 {
		t.Fatal("expected tokens, got none")
	}

	// Check first token is "分析"
	if tokens[0].Surface() != "分析" {
		t.Errorf("expected first token surface '分析', got %q", tokens[0].Surface())
	}

	// Verify Result is initialized to Surface
	for i, tok := range tokens {
		if tok.Result() != tok.Surface() {
			t.Errorf("token[%d]: expected Result=%q to equal Surface=%q", i, tok.Result(), tok.Surface())
		}
	}

	// Verify POS is populated
	for i, tok := range tokens {
		if len(tok.POS()) == 0 {
			t.Errorf("token[%d] (%q): expected POS to be populated", i, tok.Surface())
		}
	}
}

func TestTokenizeSentences(t *testing.T) {
	tk, err := New()
	if err != nil {
		t.Fatalf("failed to create tokenizer: %v", err)
	}

	input := "分析します。\n\n結果です。"
	sentences := tk.TokenizeSentences(input)

	if len(sentences) != 3 {
		t.Fatalf("expected 3 sentence groups, got %d", len(sentences))
	}

	// First sentence should have tokens
	if len(sentences[0]) == 0 {
		t.Error("expected tokens in first sentence")
	}

	// Second should be nil (empty line)
	if sentences[1] != nil {
		t.Error("expected nil for empty line")
	}

	// Third sentence should have tokens
	if len(sentences[2]) == 0 {
		t.Error("expected tokens in third sentence")
	}
}

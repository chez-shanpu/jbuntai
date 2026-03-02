package preprocess

import (
	"strings"
	"testing"
)

func TestDo_CodeBlockExtraction(t *testing.T) {
	input := "before\n```go\nfmt.Println(\"hello\")\n```\nafter"
	result, blocks := Do(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if !strings.Contains(blocks[0].Content, "fmt.Println") {
		t.Errorf("expected code block content, got %q", blocks[0].Content)
	}
	if strings.Contains(result, "fmt.Println") {
		t.Error("expected code block to be replaced with placeholder")
	}
	if !strings.Contains(result, PlaceholderPrefix) {
		t.Error("expected placeholder in result")
	}
}

func TestDo_InlineCodeExtraction(t *testing.T) {
	input := "use `go build` command"
	result, blocks := Do(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Content != "`go build`" {
		t.Errorf("expected inline code content, got %q", blocks[0].Content)
	}
	if strings.Contains(result, "`go build`") {
		t.Error("expected inline code to be replaced")
	}
}

func TestDo_EmojiRemoval(t *testing.T) {
	input := "テスト😀完了🎉"
	result, _ := Do(input)

	if strings.Contains(result, "😀") || strings.Contains(result, "🎉") {
		t.Errorf("expected emoji to be removed, got %q", result)
	}
	if !strings.Contains(result, "テスト") || !strings.Contains(result, "完了") {
		t.Errorf("expected text to be preserved, got %q", result)
	}
}

func TestDo_FullWidthNormalization(t *testing.T) {
	input := "ＡＢＣ１２３"
	result, _ := Do(input)

	if result != "ABC123" {
		t.Errorf("expected %q, got %q", "ABC123", result)
	}
}

func TestDo_FullWidthSpace(t *testing.T) {
	input := "テスト\u3000テスト"
	result, _ := Do(input)

	if result != "テスト テスト" {
		t.Errorf("expected full-width space to become half-width, got %q", result)
	}
}

func TestDo_NoChange(t *testing.T) {
	input := "普通のテキスト"
	result, blocks := Do(input)

	if len(blocks) != 0 {
		t.Errorf("expected no blocks, got %d", len(blocks))
	}
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestDo_Empty(t *testing.T) {
	result, blocks := Do("")
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
	if len(blocks) != 0 {
		t.Errorf("expected no blocks, got %d", len(blocks))
	}
}

func TestDo_Mixed(t *testing.T) {
	input := "テスト😀`code`ＡＢＣ"
	result, blocks := Do(input)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if strings.Contains(result, "😀") {
		t.Error("emoji should be removed")
	}
	if strings.Contains(result, "ＡＢＣ") {
		t.Error("full-width should be normalized")
	}
	if strings.Contains(result, "ABC") && !strings.Contains(result, "`code`") {
		// Code block should be replaced; ABC should be present
	}
}

package postprocess

import (
	"testing"

	"github.com/chez-shanpu/jbuntai/internal/preprocess"
)

func TestDo_CJKLatinSpacing(t *testing.T) {
	result := Do("テストabc日本語", nil)
	if result != "テスト abc 日本語" {
		t.Errorf("expected CJK-Latin spacing, got %q", result)
	}
}

func TestDo_NoDoubleSpaces(t *testing.T) {
	result := Do("テスト  テスト", nil)
	if result != "テスト テスト" {
		t.Errorf("expected single space, got %q", result)
	}
}

func TestDo_CodeBlockRestoration(t *testing.T) {
	blocks := []preprocess.CodeBlock{
		{Placeholder: "%%CODEBLOCK_0%%", Content: "```go\ncode\n```"},
	}
	result := Do("before %%CODEBLOCK_0%% after", blocks)
	if result != "before ```go\ncode\n``` after" {
		t.Errorf("expected code block restored, got %q", result)
	}
}

func TestDo_Empty(t *testing.T) {
	result := Do("", nil)
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestDo_LatinCJKSpacing(t *testing.T) {
	result := Do("abc日本語", nil)
	if result != "abc 日本語" {
		t.Errorf("expected Latin-CJK spacing, got %q", result)
	}
}

func TestDo_ExistingSpacePreserved(t *testing.T) {
	result := Do("テスト abc", nil)
	if result != "テスト abc" {
		t.Errorf("expected existing space preserved, got %q", result)
	}
}

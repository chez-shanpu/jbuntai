package pass

import (
	"testing"

	"github.com/chez-shanpu/jbuntai/internal/config"
)

func TestBoundaryPass_RestoreParticle(t *testing.T) {
	cfg := config.Default()
	cfg.MaxKanjiRun = 4

	bp := NewBoundaryPass(cfg)

	tokens := []Token{
		{surface: "機械", pos: []string{"名詞", "一般"}, result: "機械"},
		{surface: "学習", pos: []string{"名詞", "一般"}, result: "学習"},
		{surface: "の", pos: []string{"助詞", "連体化"}, result: "の", deleted: true},
		{surface: "結果", pos: []string{"名詞", "一般"}, result: "結果"},
		{surface: "分析", pos: []string{"名詞", "サ変接続"}, result: "分析"},
	}

	result := bp.Apply(tokens)

	// の should be restored since 機械学習結果分析 = 8 kanji > threshold
	got := renderTokens(result)
	if got != "機械学習の結果分析" {
		t.Errorf("got %q, want %q", got, "機械学習の結果分析")
	}
}

func TestBoundaryPass_NoBoundaryNeeded(t *testing.T) {
	cfg := config.Default()
	cfg.MaxKanjiRun = 5

	bp := NewBoundaryPass(cfg)

	tokens := []Token{
		{surface: "結果", pos: []string{"名詞", "一般"}, result: "結果"},
		{surface: "分析", pos: []string{"名詞", "サ変接続"}, result: "分析"},
	}

	result := bp.Apply(tokens)
	got := renderTokens(result)
	if got != "結果分析" {
		t.Errorf("got %q, want %q", got, "結果分析")
	}
}

package pass

import (
	"testing"

	"github.com/chez-shanpu/jbuntai/internal/config"
)

func TestSymbolPass_ParallelToYa(t *testing.T) {
	cfg := config.Default()
	sp := NewSymbolPass(cfg, false)

	tokens := []Token{
		{surface: "犬", pos: []string{"名詞", "一般"}, result: "犬"},
		{surface: "と", pos: []string{"助詞", "並立助詞"}, result: "と"},
		{surface: "猫", pos: []string{"名詞", "一般"}, result: "猫"},
	}

	result := sp.Apply(tokens)
	got := renderTokens(result)
	want := "犬・猫"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSymbolPass_ConjunctiveTe(t *testing.T) {
	cfg := config.Default()
	sp := NewSymbolPass(cfg, false)

	tokens := []Token{
		{surface: "調べ", pos: []string{"動詞", "自立"}, baseForm: "調べる", result: "調べ"},
		{surface: "て", pos: []string{"助詞", "接続助詞"}, result: "て"},
		{surface: "報告", pos: []string{"名詞", "サ変接続"}, result: "報告"},
	}

	result := sp.Apply(tokens)
	got := renderTokens(result)
	want := "調べ,報告"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSymbolPass_Range(t *testing.T) {
	cfg := config.Default()
	sp := NewSymbolPass(cfg, false)

	tokens := []Token{
		{surface: "1月", pos: []string{"名詞", "一般"}, result: "1月"},
		{surface: "から", pos: []string{"助詞", "格助詞"}, result: "から"},
		{surface: "3月", pos: []string{"名詞", "一般"}, result: "3月"},
		{surface: "まで", pos: []string{"助詞", "副助詞"}, result: "まで"},
	}

	result := sp.Apply(tokens)
	got := renderTokens(result)
	want := "1月〜3月〜"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

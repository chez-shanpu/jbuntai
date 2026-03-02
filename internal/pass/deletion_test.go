package pass

import (
	"testing"
)

func TestDeletionPass_RemoveHa(t *testing.T) {
	dp := NewDeletionPass()

	tokens := []Token{
		{surface: "システム", pos: []string{"名詞", "一般"}, result: "システム"},
		{surface: "は", pos: []string{"助詞", "係助詞"}, result: "は"},
		{surface: "動作", pos: []string{"名詞", "サ変接続"}, result: "動作"},
	}

	result := dp.Apply(tokens)
	got := renderTokens(result)
	want := "システム動作"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeletionPass_RemoveGa(t *testing.T) {
	dp := NewDeletionPass()

	tokens := []Token{
		{surface: "問題", pos: []string{"名詞", "一般"}, result: "問題"},
		{surface: "が", pos: []string{"助詞", "格助詞"}, result: "が"},
		{surface: "ある", pos: []string{"動詞", "自立"}, baseForm: "ある", result: "ある"},
	}

	result := dp.Apply(tokens)
	got := renderTokens(result)
	want := "問題ある"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeletionPass_KeepAdversativeGa(t *testing.T) {
	dp := NewDeletionPass()

	tokens := []Token{
		{surface: "試し", pos: []string{"動詞", "自立"}, baseForm: "試す", result: "試し"},
		{surface: "た", pos: []string{"助動詞"}, baseForm: "た", result: "た"},
		{surface: "が", pos: []string{"助詞", "格助詞"}, result: "が"},
		{surface: "、", pos: []string{"記号", "読点"}, result: "、"},
		{surface: "失敗", pos: []string{"名詞", "サ変接続"}, result: "失敗"},
	}

	result := dp.Apply(tokens)
	got := renderTokens(result)
	want := "試したが、失敗"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeletionPass_RemoveWo(t *testing.T) {
	dp := NewDeletionPass()

	tokens := []Token{
		{surface: "検討", pos: []string{"名詞", "サ変接続"}, result: "検討"},
		{surface: "を", pos: []string{"助詞", "格助詞"}, result: "を"},
		{surface: "行う", pos: []string{"動詞", "自立"}, baseForm: "行う", result: "行う"},
	}

	result := dp.Apply(tokens)
	got := renderTokens(result)
	want := "検討行う"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDeletionPass_RemoveKotoMono(t *testing.T) {
	dp := NewDeletionPass()

	tokens := []Token{
		{surface: "重要", pos: []string{"名詞", "形容動詞語幹"}, result: "重要"},
		{surface: "な", pos: []string{"助動詞"}, baseForm: "だ", result: "な"},
		{surface: "こと", pos: []string{"名詞", "非自立"}, result: "こと"},
	}

	result := dp.Apply(tokens)
	got := renderTokens(result)
	want := "重要な"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

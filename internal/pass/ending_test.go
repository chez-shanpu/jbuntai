package pass

import (
	"strings"
	"testing"
)

func TestEndingPass_RemoveDesuMasu(t *testing.T) {
	tests := []struct {
		name   string
		tokens []Token
		want   string
	}{
		{
			name: "remove ます at end",
			tokens: []Token{
				{surface: "分析", pos: []string{"名詞", "サ変接続"}, result: "分析"},
				{surface: "し", pos: []string{"動詞", "自立"}, baseForm: "する", result: "し"},
				{surface: "ます", pos: []string{"助動詞"}, baseForm: "ます", result: "ます"},
				{surface: "。", pos: []string{"記号", "句点"}, result: "。"},
			},
			want: "分析し。",
		},
		{
			name: "remove です at end",
			tokens: []Token{
				{surface: "結果", pos: []string{"名詞", "一般"}, result: "結果"},
				{surface: "です", pos: []string{"助動詞"}, baseForm: "です", result: "です"},
				{surface: "。", pos: []string{"記号", "句点"}, result: "。"},
			},
			want: "結果。",
		},
		{
			name: "remove ました at end",
			tokens: []Token{
				{surface: "行い", pos: []string{"動詞", "自立"}, baseForm: "行う", result: "行い"},
				{surface: "まし", pos: []string{"助動詞"}, baseForm: "ます", result: "まし"},
				{surface: "た", pos: []string{"助動詞"}, baseForm: "た", result: "た"},
				{surface: "。", pos: []string{"記号", "句点"}, result: "。"},
			},
			want: "行い。",
		},
		{
			name: "remove でした at end",
			tokens: []Token{
				{surface: "良好", pos: []string{"名詞", "形容動詞語幹"}, result: "良好"},
				{surface: "でし", pos: []string{"助動詞"}, baseForm: "です", result: "でし"},
				{surface: "た", pos: []string{"助動詞"}, baseForm: "た", result: "た"},
				{surface: "。", pos: []string{"記号", "句点"}, result: "。"},
			},
			want: "良好。",
		},
		{
			name: "remove である at end",
			tokens: []Token{
				{surface: "重要", pos: []string{"名詞", "形容動詞語幹"}, result: "重要"},
				{surface: "で", pos: []string{"助動詞"}, baseForm: "だ", result: "で"},
				{surface: "ある", pos: []string{"動詞", "自立"}, baseForm: "ある", result: "ある"},
				{surface: "。", pos: []string{"記号", "句点"}, result: "。"},
			},
			want: "重要。",
		},
	}

	ep := NewEndingPass()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ep.Apply(tt.tokens)
			got := renderTokens(result)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEndingPass_RemoveEndParticles(t *testing.T) {
	tests := []struct {
		name   string
		tokens []Token
		want   string
	}{
		{
			name: "remove よ",
			tokens: []Token{
				{surface: "良い", pos: []string{"形容詞", "自立"}, result: "良い"},
				{surface: "よ", pos: []string{"助詞", "終助詞"}, result: "よ"},
			},
			want: "良い",
		},
		{
			name: "remove ね",
			tokens: []Token{
				{surface: "良い", pos: []string{"形容詞", "自立"}, result: "良い"},
				{surface: "ね", pos: []string{"助詞", "終助詞"}, result: "ね"},
			},
			want: "良い",
		},
	}

	ep := NewEndingPass()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ep.Apply(tt.tokens)
			got := renderTokens(result)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func renderTokens(tokens []Token) string {
	var s strings.Builder
	for _, t := range tokens {
		if t.Deleted() {
			continue
		}
		if t.Prefix() != "" {
			s.WriteString(t.Prefix())
		}
		s.WriteString(t.Result())
	}
	return s.String()
}

package codex

import (
	"slices"
	"testing"

	"github.com/chez-shanpu/jbuntai/internal/llm"
)

func TestParseResults(t *testing.T) {
	want := []llm.DisambiguationResult{{ID: 0, Answer: "location"}}
	tests := []struct {
		name    string
		text    string
		want    []llm.DisambiguationResult
		wantErr bool
	}{
		{name: "lowercase keys", text: `[{"id":0,"answer":"location"}]`, want: want},
		{name: "capitalized keys", text: `[{"ID":0,"Answer":"location"}]`, want: want},
		{name: "code fence", text: "```json\n[{\"id\":0,\"answer\":\"location\"}]\n```", want: want},
		{name: "not json", text: "sorry", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseResults(tt.text)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

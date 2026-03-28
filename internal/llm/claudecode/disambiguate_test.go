package claudecode

import (
	"testing"

	"github.com/chez-shanpu/jbuntai/internal/llm/prompt"
)

func TestExtractJSON_PlainArray(t *testing.T) {
	input := `[{"id":0,"answer":"location"}]`
	got := prompt.ExtractJSON(input)
	if got != input {
		t.Errorf("expected %q, got %q", input, got)
	}
}

func TestExtractJSON_MarkdownFence(t *testing.T) {
	input := "```json\n[{\"id\":0,\"answer\":\"location\"}]\n```"
	want := `[{"id":0,"answer":"location"}]`
	got := prompt.ExtractJSON(input)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestExtractJSON_FenceWithoutLang(t *testing.T) {
	input := "```\n[{\"id\":0,\"answer\":\"location\"}]\n```"
	want := `[{"id":0,"answer":"location"}]`
	got := prompt.ExtractJSON(input)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestExtractJSON_SurroundingText(t *testing.T) {
	input := "Here is the result:\n[{\"id\":0,\"answer\":\"location\"}]\nDone."
	want := `[{"id":0,"answer":"location"}]`
	got := prompt.ExtractJSON(input)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestExtractJSON_Empty(t *testing.T) {
	got := prompt.ExtractJSON("")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractJSON_NoArray(t *testing.T) {
	input := "no json here"
	got := prompt.ExtractJSON(input)
	if got != input {
		t.Errorf("expected %q, got %q", input, got)
	}
}

func TestExtractJSON_WhitespaceWrapped(t *testing.T) {
	input := "  \n  [{\"id\":0,\"answer\":\"direction\"}]  \n  "
	want := `[{"id":0,"answer":"direction"}]`
	got := prompt.ExtractJSON(input)
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

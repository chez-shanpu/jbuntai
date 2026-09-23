package typesafeai

import (
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	typesafeaigo "github.com/chez-shanpu/typesafeai-go"

	"github.com/chez-shanpu/jbuntai/internal/llm"
)

func TestBuildQuestions(t *testing.T) {
	items := []llm.AmbiguousItem{
		{ID: 3, Sentence: "東京に行く", Target: "に", Position: 2, Choices: []string{"direction", "location", "other"}},
	}

	questions := buildQuestions(items)

	q, ok := questions["3"]
	if !ok {
		t.Fatalf("expected question keyed by %q, got keys: %v", "3", questions)
	}
	choiceQ, ok := q.(*typesafeaigo.ChoiceQuestion)
	if !ok {
		t.Fatalf("expected *ChoiceQuestion, got %T", q)
	}

	instructions, ok := choiceQ.Instructions.(string)
	if !ok || !strings.Contains(instructions, "東京に行く") || !strings.Contains(instructions, "に") {
		t.Fatalf("expected instructions to reference sentence and target, got: %v", choiceQ.Instructions)
	}

	if len(choiceQ.Criteria) != 3 {
		t.Fatalf("expected 3 criteria entries, got %d", len(choiceQ.Criteria))
	}
	desc, ok := choiceQ.Criteria["direction"]
	if !ok || desc == nil || *desc == "" {
		t.Fatalf("expected non-empty description for %q, got %v", "direction", desc)
	}
}

func TestBuildQuestions_UnknownChoiceFallsBackToNil(t *testing.T) {
	items := []llm.AmbiguousItem{
		{ID: 1, Sentence: "s", Target: "が", Position: 0, Choices: []string{"unknown"}},
	}

	questions := buildQuestions(items)
	choiceQ := questions["1"].(*typesafeaigo.ChoiceQuestion)

	desc, ok := choiceQ.Criteria["unknown"]
	if !ok || desc != nil {
		t.Fatalf("expected nil description for unmapped particle/choice, got %v", desc)
	}
}

func TestParseResults(t *testing.T) {
	items := []llm.AmbiguousItem{
		{ID: 1, Choices: []string{"direction", "location"}},
		{ID: 2, Choices: []string{"location", "means"}},
	}

	resp := &typesafeaigo.SystemOneResponse{
		Answers: typesafeaigo.QuestionKeyToAnswer{
			"1": &typesafeaigo.ChoiceAnswer{Choice: "direction"},
			// "2" intentionally missing to exercise the skip path.
		},
	}

	results := parseResults(resp, items)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0].ID != 1 || results[0].Answer != "direction" {
		t.Fatalf("unexpected result: %+v", results[0])
	}
}

func TestParseResults_SkipsWrongAnswerType(t *testing.T) {
	items := []llm.AmbiguousItem{
		{ID: 1, Choices: []string{"direction"}},
	}

	resp := &typesafeaigo.SystemOneResponse{
		Answers: typesafeaigo.QuestionKeyToAnswer{
			"1": &typesafeaigo.NoulAnswer{Noul: 0.9},
		},
	}

	results := parseResults(resp, items)

	if len(results) != 0 {
		t.Fatalf("expected 0 results for mismatched answer type, got: %v", results)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newTestDisambiguator(rt roundTripFunc) *disambiguatorImpl {
	return &disambiguatorImpl{
		client: newClient("test-key", &http.Client{Transport: rt}),
		model:  "jev-test",
	}
}

func TestDisambiguate(t *testing.T) {
	items := []llm.AmbiguousItem{
		{ID: 0, Sentence: "東京に行く", Target: "に", Position: 1, Choices: []string{"direction", "location", "time", "other"}},
	}
	tests := []struct {
		name       string
		status     int
		body       string
		want       []llm.DisambiguationResult
		wantErr    bool
		wantAPIErr bool
	}{
		{
			name:   "success",
			status: http.StatusOK,
			body:   `{"model":"jev-test","answers":{"0":{"type":"choice","choice":"direction","probabilities":{},"confidence":0.9}}}`,
			want:   []llm.DisambiguationResult{{ID: 0, Answer: "direction"}},
		},
		{name: "null body", status: http.StatusOK, body: `null`, wantErr: true},
		{name: "api error", status: http.StatusInternalServerError, body: `{"error":"boom"}`, wantErr: true, wantAPIErr: true},
		{name: "malformed json", status: http.StatusOK, body: `{`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newTestDisambiguator(func(r *http.Request) (*http.Response, error) {
				if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
					t.Errorf("Authorization = %q", got)
				}
				var req struct {
					Model     string         `json:"model"`
					Questions map[string]any `json:"questions"`
				}
				if err := json.UnmarshalRead(r.Body, &req); err != nil {
					t.Errorf("decode request: %v", err)
				}
				if req.Model != "jev-test" || req.Questions["0"] == nil {
					t.Errorf("unexpected request: %+v", req)
				}
				return &http.Response{
					StatusCode: tt.status,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			})

			got, err := d.Disambiguate(t.Context(), items)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got results %v", got)
				}
				var apiErr *typesafeaigo.APIError
				if tt.wantAPIErr && !errors.As(err, &apiErr) {
					t.Errorf("expected *APIError in chain, got %v", err)
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

func TestDisambiguate_EmptyItemsSkipsRequest(t *testing.T) {
	d := newTestDisambiguator(func(*http.Request) (*http.Response, error) {
		t.Error("HTTP must not be called for empty items")
		return nil, errors.New("unexpected request")
	})
	got, err := d.Disambiguate(t.Context(), nil)
	if got != nil || err != nil {
		t.Fatalf("got (%v, %v), want (nil, nil)", got, err)
	}
}

package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/chez-shanpu/jbuntai/internal/llm"
)

func TestDisambiguator_Success(t *testing.T) {
	d := &Disambiguator{
		Results: []llm.DisambiguationResult{
			{ID: 1, Answer: "direction"},
		},
	}

	items := []llm.AmbiguousItem{
		{ID: 1, Sentence: "東京に行く", Target: "に", Position: 2, Choices: []string{"direction", "location"}},
	}

	results, err := d.Disambiguate(context.Background(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.Called {
		t.Fatal("expected Called to be true")
	}
	if len(d.CalledWith) != 1 || d.CalledWith[0].ID != 1 {
		t.Fatalf("unexpected CalledWith: %v", d.CalledWith)
	}
	if len(results) != 1 || results[0].Answer != "direction" {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestDisambiguator_Error(t *testing.T) {
	d := &Disambiguator{
		Err: errors.New("api error"),
	}

	_, err := d.Disambiguate(context.Background(), nil)
	if err == nil || err.Error() != "api error" {
		t.Fatalf("expected api error, got: %v", err)
	}
	if !d.Called {
		t.Fatal("expected Called to be true")
	}
}

func TestFinisher_Success(t *testing.T) {
	f := &Finisher{
		FinishedText: "refined text",
	}

	result, err := f.Finish(context.Background(), "original", "transformed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Called {
		t.Fatal("expected Called to be true")
	}
	if f.CalledWith[0] != "original" || f.CalledWith[1] != "transformed" {
		t.Fatalf("unexpected CalledWith: %v", f.CalledWith)
	}
	if result != "refined text" {
		t.Fatalf("unexpected result: %s", result)
	}
}

func TestFinisher_Error(t *testing.T) {
	f := &Finisher{
		Err: errors.New("finish error"),
	}

	_, err := f.Finish(context.Background(), "original", "transformed")
	if err == nil || err.Error() != "finish error" {
		t.Fatalf("expected finish error, got: %v", err)
	}
	if !f.Called {
		t.Fatal("expected Called to be true")
	}
}

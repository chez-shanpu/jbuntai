package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/chez-shanpu/jbuntai/internal/config"
	mocklm "github.com/chez-shanpu/jbuntai/internal/llm/mock"
	"github.com/chez-shanpu/jbuntai/internal/pass"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name   string
		tokens []pass.Token
		want   string
	}{
		{
			name: "basic tokens",
			tokens: func() []pass.Token {
				t1 := pass.NewToken("東京", nil, "", "")
				t2 := pass.NewToken("に", nil, "", "")
				t2.SetPrefix(">")
				t3 := pass.NewToken("行く", nil, "", "")
				return []pass.Token{t1, t2, t3}
			}(),
			want: "東京>に行く",
		},
		{
			name: "deleted token skipped",
			tokens: func() []pass.Token {
				t1 := pass.NewToken("東京", nil, "", "")
				t2 := pass.NewToken("は", nil, "", "")
				t2.SetDeleted(true)
				t3 := pass.NewToken("広い", nil, "", "")
				return []pass.Token{t1, t2, t3}
			}(),
			want: "東京広い",
		},
		{
			name:   "empty tokens",
			tokens: nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Render(tt.tokens)
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPipeline_LLMFinisher_Success(t *testing.T) {
	cfg := config.Default()
	cfg.LLM.Finish = new(true)

	d := &mocklm.Disambiguator{}
	f := &mocklm.Finisher{
		FinishedText: "LLM refined text",
	}

	p, err := New(cfg, true, WithFinisher(f), WithDisambiguator(d))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}
	result := p.Run(context.Background(), "東京に行きます。")

	if !f.Called {
		t.Fatal("expected Finisher to be called")
	}
	if result != "LLM refined text" {
		t.Errorf("expected LLM result, got: %q", result)
	}
}

func TestPipeline_LLMFinisher_Error_Fallback(t *testing.T) {
	cfg := config.Default()
	cfg.LLM.Finish = new(true)

	d := &mocklm.Disambiguator{}
	f := &mocklm.Finisher{
		Err: errors.New("api failure"),
	}

	p, err := New(cfg, true, WithFinisher(f), WithDisambiguator(d))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}
	result := p.Run(context.Background(), "東京に行きます。")

	if !f.Called {
		t.Fatal("expected Finisher to be called")
	}
	// Should fallback to rule-based result (non-empty)
	if result == "" {
		t.Error("expected non-empty fallback result")
	}
	if result == "LLM refined text" {
		t.Error("should not return LLM text on error")
	}
}

func TestPipeline_LLMOff_FinisherNotCalled(t *testing.T) {
	cfg := config.Default()
	cfg.LLM.Finish = new(true)

	d := &mocklm.Disambiguator{}
	f := &mocklm.Finisher{
		FinishedText: "should not appear",
	}

	p, err := New(cfg, false, WithFinisher(f), WithDisambiguator(d))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}
	result := p.Run(context.Background(), "東京に行きます。")

	if f.Called {
		t.Fatal("Finisher should not be called when llmOn=false")
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestPipeline_Disambiguator_Success(t *testing.T) {
	cfg := config.Default()
	cfg.LLM.Disambiguate = new(true)
	cfg.LLM.Finish = new(false)

	d := &mocklm.Disambiguator{}
	f := &mocklm.Finisher{}

	p, err := New(cfg, true, WithDisambiguator(d), WithFinisher(f))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}
	_ = p.Run(context.Background(), "東京で会議をする。")

	if !d.Called {
		t.Fatal("expected Disambiguator to be called")
	}
	if len(d.CalledWith) == 0 {
		t.Fatal("expected Disambiguator to be called with items")
	}
}

func TestPipeline_Disambiguator_Error_Fallback(t *testing.T) {
	cfg := config.Default()
	cfg.LLM.Disambiguate = new(true)
	cfg.LLM.Finish = new(false)

	d := &mocklm.Disambiguator{
		Err: errors.New("disambiguate failure"),
	}
	f := &mocklm.Finisher{}

	p, err := New(cfg, true, WithDisambiguator(d), WithFinisher(f))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}
	result := p.Run(context.Background(), "東京で会議をする。")

	if !d.Called {
		t.Fatal("expected Disambiguator to be called")
	}
	// Should fallback to heuristic-based result (non-empty)
	if result == "" {
		t.Error("expected non-empty fallback result")
	}
}

func TestPipeline_Disambiguator_NotCalled_WhenOff(t *testing.T) {
	cfg := config.Default()
	cfg.LLM.Disambiguate = new(false)
	cfg.LLM.Finish = new(false)

	d := &mocklm.Disambiguator{}
	f := &mocklm.Finisher{}

	p, err := New(cfg, true, WithDisambiguator(d), WithFinisher(f))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}
	_ = p.Run(context.Background(), "東京で会議をする。")

	if d.Called {
		t.Fatal("Disambiguator should not be called when disambiguate is disabled")
	}
}

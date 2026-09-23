package llm

import (
	"strings"
	"testing"

	"github.com/chez-shanpu/jbuntai/internal/config"
)

func init() {
	RegisterDisambiguator("test-disambiguate-only", func(*config.Config) (Disambiguator, error) { return nil, nil })
}

func TestNewFinisher_Errors(t *testing.T) {
	tests := []struct {
		backend string
		want    string
	}{
		{"test-disambiguate-only", `LLM backend "test-disambiguate-only" does not support finishing`},
		{"nonexistent", `unknown LLM backend "nonexistent"`},
	}
	for _, tt := range tests {
		t.Run(tt.backend, func(t *testing.T) {
			cfg := config.Default()
			cfg.LLM.Backend = tt.backend
			_, err := NewFinisher(cfg)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("NewFinisher() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRegisterDisambiguator_PanicsOnDuplicate(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	RegisterDisambiguator("test-disambiguate-only", func(*config.Config) (Disambiguator, error) { return nil, nil })
}

package llm

import (
	"fmt"

	"github.com/chez-shanpu/jbuntai/internal/config"
)

// DisambiguatorFactory creates a Disambiguator for the given backend.
type DisambiguatorFactory func(cfg *config.Config) (Disambiguator, error)

// FinisherFactory creates a Finisher for the given backend.
type FinisherFactory func(cfg *config.Config) (Finisher, error)

var (
	disambiguatorFactories = map[string]DisambiguatorFactory{}
	finisherFactories      = map[string]FinisherFactory{}
)

// RegisterDisambiguator registers a Disambiguator factory for the given backend name.
func RegisterDisambiguator(name string, factory DisambiguatorFactory) {
	disambiguatorFactories[name] = factory
}

// RegisterFinisher registers a Finisher factory for the given backend name.
func RegisterFinisher(name string, factory FinisherFactory) {
	finisherFactories[name] = factory
}

// NewDisambiguator creates a Disambiguator based on the configured backend.
func NewDisambiguator(cfg *config.Config) (Disambiguator, error) {
	factory, ok := disambiguatorFactories[cfg.LLM.Backend]
	if !ok {
		return nil, fmt.Errorf("unknown LLM backend: %q", cfg.LLM.Backend)
	}
	return factory(cfg)
}

// NewFinisher creates a Finisher based on the configured backend.
func NewFinisher(cfg *config.Config) (Finisher, error) {
	factory, ok := finisherFactories[cfg.LLM.Backend]
	if !ok {
		return nil, fmt.Errorf("unknown LLM backend: %q", cfg.LLM.Backend)
	}
	return factory(cfg)
}

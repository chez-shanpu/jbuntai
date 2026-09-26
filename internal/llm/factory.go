package llm

import (
	"fmt"
	"maps"
	"slices"
	"strings"

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
// It panics if a factory is already registered for the name.
func RegisterDisambiguator(name string, factory DisambiguatorFactory) {
	if _, dup := disambiguatorFactories[name]; dup {
		panic("llm: RegisterDisambiguator called twice for backend " + name)
	}
	disambiguatorFactories[name] = factory
}

// RegisterFinisher registers a Finisher factory for the given backend name.
// It panics if a factory is already registered for the name.
func RegisterFinisher(name string, factory FinisherFactory) {
	if _, dup := finisherFactories[name]; dup {
		panic("llm: RegisterFinisher called twice for backend " + name)
	}
	finisherFactories[name] = factory
}

// DisambiguatorBackends returns the sorted names of backends that support disambiguation.
func DisambiguatorBackends() []string {
	return slices.Sorted(maps.Keys(disambiguatorFactories))
}

// FinisherBackends returns the sorted names of backends that support finishing.
func FinisherBackends() []string {
	return slices.Sorted(maps.Keys(finisherFactories))
}

// NewDisambiguator creates a Disambiguator based on the configured backend.
func NewDisambiguator(cfg *config.Config) (Disambiguator, error) {
	backend := cfg.LLM.EffectiveDisambiguateBackend()
	factory, ok := disambiguatorFactories[backend]
	if !ok {
		return nil, unsupportedBackendError(backend, "disambiguation", DisambiguatorBackends())
	}
	return factory(cfg)
}

// NewFinisher creates a Finisher based on the configured backend.
func NewFinisher(cfg *config.Config) (Finisher, error) {
	backend := cfg.LLM.Backend
	factory, ok := finisherFactories[backend]
	if !ok {
		return nil, unsupportedBackendError(backend, "finishing", FinisherBackends())
	}
	return factory(cfg)
}

// unsupportedBackendError distinguishes a backend that is not registered at
// all from one that is registered but does not support the given stage.
func unsupportedBackendError(backend, stage string, available []string) error {
	_, isDisambiguator := disambiguatorFactories[backend]
	_, isFinisher := finisherFactories[backend]
	if isDisambiguator || isFinisher {
		return fmt.Errorf("LLM backend %q does not support %s (available: %s)", backend, stage, strings.Join(available, ", "))
	}
	return fmt.Errorf("unknown LLM backend %q (available: %s)", backend, strings.Join(available, ", "))
}

package mock

import (
	"context"
	"sync"

	"github.com/chez-shanpu/jbuntai/internal/llm"
)

// Disambiguator is a mock implementation of llm.Disambiguator.
type Disambiguator struct {
	mu         sync.Mutex
	Results    []llm.DisambiguationResult
	Err        error
	Called     bool
	CalledWith []llm.AmbiguousItem
}

// Disambiguate returns the pre-configured results or error.
func (m *Disambiguator) Disambiguate(_ context.Context, items []llm.AmbiguousItem) ([]llm.DisambiguationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Called = true
	m.CalledWith = items
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Results, nil
}

// Finisher is a mock implementation of llm.Finisher.
type Finisher struct {
	mu           sync.Mutex
	FinishedText string
	Err          error
	Called       bool
	CalledWith   [2]string // [original, transformed]
}

// Finish returns the pre-configured text or error.
func (m *Finisher) Finish(_ context.Context, original, transformed string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Called = true
	m.CalledWith = [2]string{original, transformed}
	if m.Err != nil {
		return "", m.Err
	}
	return m.FinishedText, nil
}

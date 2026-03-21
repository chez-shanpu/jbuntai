package llm

import "context"

// Disambiguator resolves ambiguous particle classifications.
type Disambiguator interface {
	Disambiguate(ctx context.Context, items []AmbiguousItem) ([]DisambiguationResult, error)
}

// Finisher performs final text restructuring via LLM.
type Finisher interface {
	Finish(ctx context.Context, original, transformed string) (string, error)
}

// AmbiguousItem represents a particle that needs disambiguation.
type AmbiguousItem struct {
	ID       int      `json:"id"`
	Sentence string   `json:"sentence"`
	Target   string   `json:"target"`
	Position int      `json:"position"`
	Choices  []string `json:"choices"`
}

// DisambiguationResult holds the disambiguation answer.
type DisambiguationResult struct {
	ID     int    `json:"id"`
	Answer string `json:"answer"`
}

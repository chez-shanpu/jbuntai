package codex

import (
	"github.com/chez-shanpu/jbuntai/internal/config"
	"github.com/chez-shanpu/jbuntai/internal/llm"
)

const (
	defaultDisambiguateModel     = "gpt-6-luna"
	defaultFinishModel           = "gpt-6-sol"
	defaultDisambiguateReasoning = "none"
	defaultFinishReasoning       = "low"
)

func init() {
	llm.RegisterDisambiguator(config.BackendCodex, func(cfg *config.Config) (llm.Disambiguator, error) {
		model := cfg.LLM.DisambiguateModel
		if model == "" {
			model = defaultDisambiguateModel
		}
		reasoning := cfg.LLM.DisambiguateReasoning
		if reasoning == "" {
			reasoning = defaultDisambiguateReasoning
		}
		return &disambiguatorImpl{client: newClient(), model: model, reasoningEffort: reasoning}, nil
	})
	llm.RegisterFinisher(config.BackendCodex, func(cfg *config.Config) (llm.Finisher, error) {
		model := cfg.LLM.FinishModel
		if model == "" {
			model = defaultFinishModel
		}
		reasoning := cfg.LLM.FinishReasoning
		if reasoning == "" {
			reasoning = defaultFinishReasoning
		}
		return &finisherImpl{client: newClient(), model: model, reasoningEffort: reasoning}, nil
	})
}

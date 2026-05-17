package codex

import (
	"sync"

	"github.com/chez-shanpu/jbuntai/internal/config"
	"github.com/chez-shanpu/jbuntai/internal/llm"
)

const (
	defaultDisambiguateModel     = "gpt-5.4-mini"
	defaultFinishModel           = "gpt-5.4"
	defaultDisambiguateReasoning = "none"
	defaultFinishReasoning       = "low"
)

var (
	sharedClient     *client
	sharedClientOnce sync.Once
	sharedClientErr  error
)

func getSharedClient() (*client, error) {
	sharedClientOnce.Do(func() {
		sharedClient, sharedClientErr = newClient()
	})
	return sharedClient, sharedClientErr
}

func init() {
	llm.RegisterDisambiguator(config.BackendCodex, func(cfg *config.Config) (llm.Disambiguator, error) {
		client, err := getSharedClient()
		if err != nil {
			return nil, err
		}
		model := cfg.LLM.DisambiguateModel
		if model == "" {
			model = defaultDisambiguateModel
		}
		reasoning := cfg.LLM.DisambiguateReasoning
		if reasoning == "" {
			reasoning = defaultDisambiguateReasoning
		}
		return &disambiguatorImpl{client: client, model: model, reasoningEffort: reasoning}, nil
	})
	llm.RegisterFinisher(config.BackendCodex, func(cfg *config.Config) (llm.Finisher, error) {
		client, err := getSharedClient()
		if err != nil {
			return nil, err
		}
		model := cfg.LLM.FinishModel
		if model == "" {
			model = defaultFinishModel
		}
		reasoning := cfg.LLM.FinishReasoning
		if reasoning == "" {
			reasoning = defaultFinishReasoning
		}
		return &finisherImpl{client: client, model: model, reasoningEffort: reasoning}, nil
	})
}

package claudecode

import (
	"sync"

	"github.com/chez-shanpu/jbuntai/internal/config"
	"github.com/chez-shanpu/jbuntai/internal/llm"
)

const (
	defaultDisambiguateModel = "haiku"
	defaultFinishModel       = "sonnet"
)

var (
	sharedClient     *client
	sharedClientOnce sync.Once
)

func getSharedClient() *client {
	sharedClientOnce.Do(func() {
		sharedClient = newClient()
	})
	return sharedClient
}

func init() {
	llm.RegisterDisambiguator(config.BackendClaudeCode, func(cfg *config.Config) (llm.Disambiguator, error) {
		model := cfg.LLM.DisambiguateModel
		if model == "" {
			model = defaultDisambiguateModel
		}
		return &disambiguatorImpl{client: getSharedClient(), model: model}, nil
	})
	llm.RegisterFinisher(config.BackendClaudeCode, func(cfg *config.Config) (llm.Finisher, error) {
		model := cfg.LLM.FinishModel
		if model == "" {
			model = defaultFinishModel
		}
		return &finisherImpl{client: getSharedClient(), model: model}, nil
	})
}

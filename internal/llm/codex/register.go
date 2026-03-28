package codex

import (
	"sync"

	"github.com/chez-shanpu/jbuntai/internal/config"
	"github.com/chez-shanpu/jbuntai/internal/llm"
)

const (
	defaultDisambiguateModel = "gpt-5.4-mini"
	defaultFinishModel       = "gpt-5.4"
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
		return &disambiguatorImpl{client: client, model: model}, nil
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
		return &finisherImpl{client: client, model: model}, nil
	})
}

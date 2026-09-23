package typesafeai

import (
	typesafeaigo "github.com/chez-shanpu/typesafeai-go"

	"github.com/chez-shanpu/jbuntai/internal/config"
	"github.com/chez-shanpu/jbuntai/internal/llm"
)

func init() {
	llm.RegisterDisambiguator(config.BackendTypesafeAI, func(cfg *config.Config) (llm.Disambiguator, error) {
		c, err := newClientFromEnv()
		if err != nil {
			return nil, err
		}
		model := cfg.LLM.DisambiguateModel
		if model == "" {
			model = typesafeaigo.ModelJevLatest
		}
		return &disambiguatorImpl{client: c, model: model}, nil
	})
	// Finisher is intentionally not registered: the System One API only
	// produces typed Noul/Choice/Score answers, not free-form text suitable
	// for the Finish step.
}

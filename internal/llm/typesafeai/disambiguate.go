package typesafeai

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	typesafeaigo "github.com/chez-shanpu/typesafeai-go"

	"github.com/chez-shanpu/jbuntai/internal/llm"
	"github.com/chez-shanpu/jbuntai/internal/particle"
)

// disambiguatorImpl implements the Disambiguator interface using the
// TypeSafe AI System One API's jev model.
type disambiguatorImpl struct {
	client *client
	model  string
}

// Disambiguate resolves ambiguous particles using a single System One
// request containing one ChoiceQuestion per item.
func (d *disambiguatorImpl) Disambiguate(ctx context.Context, items []llm.AmbiguousItem) ([]llm.DisambiguationResult, error) {
	if len(items) == 0 {
		return nil, nil
	}

	resp, err := d.client.systemOne.Do(ctx, &typesafeaigo.SystemOneRequest{
		Model:     d.model,
		Questions: buildQuestions(items),
	})
	if err != nil {
		return nil, fmt.Errorf("disambiguate failed: %w", err)
	}
	// Do returns (nil, nil) when the response body is JSON null.
	if resp == nil {
		return nil, errors.New("disambiguate failed: empty response")
	}

	return parseResults(resp, items), nil
}

// buildQuestions converts each AmbiguousItem into a ChoiceQuestion, keyed by
// its ID (as a string) so responses can be mapped back unambiguously.
func buildQuestions(items []llm.AmbiguousItem) map[string]typesafeaigo.Question {
	questions := make(map[string]typesafeaigo.Question, len(items))
	for _, item := range items {
		questions[strconv.Itoa(item.ID)] = &typesafeaigo.ChoiceQuestion{
			Instructions: fmt.Sprintf(
				"In the following Japanese sentence, classify the grammatical function of the particle %q at token position %d:\n%s",
				item.Target, item.Position, item.Sentence,
			),
			Criteria: criteriaFor(item.Target, item.Choices),
		}
	}
	return questions
}

// parseResults maps the API response back onto DisambiguationResults, one
// per item that has a well-formed ChoiceAnswer. Items with a missing or
// malformed answer are silently skipped rather than failing the whole call.
func parseResults(resp *typesafeaigo.SystemOneResponse, items []llm.AmbiguousItem) []llm.DisambiguationResult {
	results := make([]llm.DisambiguationResult, 0, len(items))
	for _, item := range items {
		ans, ok := resp.Answers[strconv.Itoa(item.ID)]
		if !ok {
			continue
		}
		choice, ok := ans.(*typesafeaigo.ChoiceAnswer)
		if !ok {
			continue
		}
		results = append(results, llm.DisambiguationResult{
			ID:     item.ID,
			Answer: choice.Choice,
		})
	}
	return results
}

// criteriaFor builds a ChoiceQuestion.Criteria map for the given particle
// surface and choice labels. Choices without a known description map to nil
// (no description).
func criteriaFor(surface string, choices []string) map[string]*string {
	criteria := make(map[string]*string, len(choices))
	for _, c := range choices {
		criteria[c] = nil
		if d, ok := particle.Description(surface, c); ok {
			criteria[c] = &d
		}
	}
	return criteria
}

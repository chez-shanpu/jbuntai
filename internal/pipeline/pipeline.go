package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/chez-shanpu/jbuntai/internal/config"
	"github.com/chez-shanpu/jbuntai/internal/llm"
	"github.com/chez-shanpu/jbuntai/internal/pass"
	"github.com/chez-shanpu/jbuntai/internal/tokenizer"
)

// Pipeline orchestrates the transformation passes.
type Pipeline struct {
	tokenizer     *tokenizer.Tokenizer
	passes        []pass.Pass
	symbolPass    *pass.SymbolPass // Direct reference for disambiguation
	cfg           *config.Config
	llmOn         bool
	finisher      llm.Finisher
	disambiguator llm.Disambiguator
	logger        *slog.Logger
}

// Option configures the Pipeline.
type Option func(*Pipeline)

// WithFinisher sets an external Finisher (useful for testing).
func WithFinisher(f llm.Finisher) Option {
	return func(p *Pipeline) {
		p.finisher = f
	}
}

// WithDisambiguator sets an external Disambiguator (useful for testing).
func WithDisambiguator(d llm.Disambiguator) Option {
	return func(p *Pipeline) {
		p.disambiguator = d
	}
}

// WithLogger sets the logger for debug output.
func WithLogger(l *slog.Logger) Option {
	return func(p *Pipeline) {
		p.logger = l
	}
}

// New creates a new Pipeline with the configured passes.
func New(cfg *config.Config, llmOn bool, opts ...Option) (*Pipeline, error) {
	p := &Pipeline{
		cfg:   cfg,
		llmOn: llmOn,
	}
	for _, o := range opts {
		o(p)
	}

	// Set default logger if not injected
	if p.logger == nil {
		p.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	p.logger.Debug("creating tokenizer")
	tk, err := tokenizer.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create tokenizer: %w", err)
	}
	p.tokenizer = tk

	p.logger.Debug("building passes")
	passes, symbolPass := buildPasses(cfg, llmOn)
	p.passes = passes
	p.symbolPass = symbolPass

	// Set up LLM components if enabled and not already injected
	p.logger.Debug("setting up LLM components")
	if p.llmOn && p.finisher == nil {
		f, err := llm.NewFinisher(p.cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create finisher: %w", err)
		}
		p.finisher = f
	}
	if p.llmOn && p.disambiguator == nil {
		d, err := llm.NewDisambiguator(p.cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create disambiguator: %w", err)
		}
		p.disambiguator = d
	}

	return p, nil
}

// Run executes the full pipeline on the input text.
func (p *Pipeline) Run(ctx context.Context, text string) string {
	p.logger.Debug("tokenizing sentences")
	tokenizedLines := p.tokenizer.TokenizeSentences(text)
	rawLines := strings.Split(text, "\n")

	// Disambiguate particles if enabled
	var disambiguations map[int]map[int]string
	if p.llmOn && p.disambiguator != nil && p.cfg.LLM.IsDisambiguateEnabled() {
		p.logger.Debug("disambiguating particles")
		disambiguations = p.disambiguateParticles(ctx, rawLines, tokenizedLines)
	}

	// Apply passes line by line
	p.logger.Debug("applying passes", "lines", len(tokenizedLines))
	var resultLines []string
	for i, tokens := range tokenizedLines {
		if disambiguations != nil && p.symbolPass != nil {
			p.symbolPass.SetDisambiguations(disambiguations[i])
		}
		resultLines = append(resultLines, p.processLine(tokens, rawLines, i))
	}

	result := strings.Join(resultLines, "\n")

	p.logger.Debug("applying finisher")
	return p.applyFinisher(ctx, text, result)
}

// disambiguateParticles collects ambiguous particles from all lines and calls the Disambiguator once.
func (p *Pipeline) disambiguateParticles(ctx context.Context, rawLines []string, tokenizedLines [][]pass.Token) map[int]map[int]string {
	var items []llm.AmbiguousItem
	type itemKey struct {
		lineIdx  int
		tokenIdx int
	}
	var keys []itemKey

	for lineIdx, tokens := range tokenizedLines {
		if tokens == nil {
			continue
		}
		sentence := ""
		if lineIdx < len(rawLines) {
			sentence = rawLines[lineIdx]
		}
		for tokenIdx, t := range tokens {
			if t.Deleted() || !t.IsParticle() {
				continue
			}
			var choices []string
			switch {
			case t.Surface() == "で" && t.POSSub1() == "格助詞":
				choices = []string{"location", "means", "other"}
			case t.Surface() == "に" && t.POSSub1() == "格助詞":
				choices = []string{"direction", "location", "time", "other"}
			case t.Surface() == "と" && t.POSSub1() == "格助詞":
				choices = []string{"quotation", "parallel", "companion", "other"}
			}
			if len(choices) == 0 {
				continue
			}
			items = append(items, llm.AmbiguousItem{
				ID:       len(items),
				Sentence: sentence,
				Target:   t.Surface(),
				Position: tokenIdx,
				Choices:  choices,
			})
			keys = append(keys, itemKey{lineIdx: lineIdx, tokenIdx: tokenIdx})
		}
	}

	if len(items) == 0 {
		return nil
	}

	p.logger.Debug("collected ambiguous items", "count", len(items))
	results, err := p.disambiguator.Disambiguate(ctx, items)
	if err != nil {
		p.logger.Warn("LLM disambiguate failed, using heuristics", "error", err)
		return nil
	}

	// Build map[lineIdx]map[tokenIdx]answer
	p.logger.Debug("building disambiguation map", "results", len(results))
	disambiguations := make(map[int]map[int]string)
	for _, r := range results {
		if r.ID < 0 || r.ID >= len(keys) {
			continue
		}
		k := keys[r.ID]
		if disambiguations[k.lineIdx] == nil {
			disambiguations[k.lineIdx] = make(map[int]string)
		}
		disambiguations[k.lineIdx][k.tokenIdx] = r.Answer
	}

	return disambiguations
}

// applyFinisher applies the LLM finisher if enabled, falling back to the rule-based result on error.
func (p *Pipeline) applyFinisher(ctx context.Context, original, result string) string {
	if !p.llmOn || p.finisher == nil || !p.cfg.LLM.IsFinishEnabled() {
		return result
	}
	p.logger.Debug("calling LLM finisher")
	finished, err := p.finisher.Finish(ctx, original, result)
	if err != nil {
		p.logger.Warn("LLM finish failed, using rule-based result", "error", err)
		return result
	}
	return finished
}

// processLine transforms a single line.
// If tokens is nil (non-tokenizable line), the original raw line is preserved.
func (p *Pipeline) processLine(tokens []pass.Token, rawLines []string, index int) string {
	if tokens == nil {
		if index < len(rawLines) {
			return rawLines[index]
		}
		return ""
	}
	for _, ps := range p.passes {
		tokens = ps.Apply(tokens)
	}
	return Render(tokens)
}

// Render converts tokens back to a string.
func Render(tokens []pass.Token) string {
	var sb strings.Builder
	for _, t := range tokens {
		if t.Deleted() {
			continue
		}
		if t.Prefix() != "" {
			sb.WriteString(t.Prefix())
		}
		sb.WriteString(t.Result())
	}
	return sb.String()
}

// buildPasses creates the ordered list of transformation passes and returns both the pass list and a direct reference to SymbolPass.
func buildPasses(cfg *config.Config, llmOn bool) ([]pass.Pass, *pass.SymbolPass) {
	symbolPass := pass.NewSymbolPass(cfg, llmOn)
	var passes []pass.Pass
	passes = append(passes, pass.NewEndingPass())
	passes = append(passes, symbolPass)
	passes = append(passes, pass.NewDeletionPass())
	passes = append(passes, pass.NewBoundaryPass(cfg))
	return passes, symbolPass
}

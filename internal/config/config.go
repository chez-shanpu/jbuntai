package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Backend name constants.
const (
	BackendClaudeCode = "claudecode"
	BackendCodex      = "codex"
)

// Config holds the application configuration.
type Config struct {
	MaxKanjiRun int       `yaml:"max_kanji_run"`
	LLM         LLMConfig `yaml:"llm"`
}

// LLMConfig holds LLM-related configuration.
type LLMConfig struct {
	Backend               string `yaml:"backend"` // "claudecode" (default) or "codex"
	DisambiguateModel     string `yaml:"disambiguate_model"`
	DisambiguateReasoning string `yaml:"disambiguate_reasoning"`
	FinishModel           string `yaml:"finish_model"`
	FinishReasoning       string `yaml:"finish_reasoning"`
	Disambiguate          *bool  `yaml:"disambiguate"`
	Finish                *bool  `yaml:"finish"`
}

// IsDisambiguateEnabled returns whether disambiguation is enabled (default: true).
func (c LLMConfig) IsDisambiguateEnabled() bool {
	if c.Disambiguate == nil {
		return true
	}
	return *c.Disambiguate
}

// IsFinishEnabled returns whether finishing is enabled (default: true).
func (c LLMConfig) IsFinishEnabled() bool {
	if c.Finish == nil {
		return true
	}
	return *c.Finish
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		MaxKanjiRun: 5,
		LLM: LLMConfig{
			Backend:      BackendCodex,
			Disambiguate: new(true),
			Finish:       new(true),
		},
	}
}

// Load reads configuration from the given path, or the default path.
func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return cfg, nil
		}
		path = filepath.Join(home, ".config", "jbuntai", "config.yaml")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for zero values
	if cfg.MaxKanjiRun <= 0 {
		cfg.MaxKanjiRun = 5
	}
	if cfg.LLM.Backend == "" {
		cfg.LLM.Backend = BackendCodex
	}
	return cfg, nil
}

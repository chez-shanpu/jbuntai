package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.MaxKanjiRun != 5 {
		t.Errorf("expected MaxKanjiRun=5, got %d", cfg.MaxKanjiRun)
	}
	// Model defaults are handled by each backend package, not config
	if cfg.LLM.DisambiguateModel != "" {
		t.Errorf("expected DisambiguateModel empty, got %q", cfg.LLM.DisambiguateModel)
	}
	if cfg.LLM.FinishModel != "" {
		t.Errorf("expected FinishModel empty, got %q", cfg.LLM.FinishModel)
	}
	if !cfg.LLM.IsDisambiguateEnabled() {
		t.Error("expected Disambiguate=true by default")
	}
	if !cfg.LLM.IsFinishEnabled() {
		t.Error("expected Finish=true by default")
	}
}

func TestLoad_FileNotExist(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxKanjiRun != 5 {
		t.Errorf("expected default MaxKanjiRun, got %d", cfg.MaxKanjiRun)
	}
}

func TestLoad_EmptyLLMSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("llm:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bool fields should retain default true when YAML section is empty
	if !cfg.LLM.IsDisambiguateEnabled() {
		t.Error("expected Disambiguate to remain true with empty llm section")
	}
	if !cfg.LLM.IsFinishEnabled() {
		t.Error("expected Finish to remain true with empty llm section")
	}
	// Model fields remain empty (defaults applied by backend packages)
	if cfg.LLM.DisambiguateModel != "" {
		t.Errorf("expected DisambiguateModel empty, got %q", cfg.LLM.DisambiguateModel)
	}
}

func TestLoad_ExplicitFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := "llm:\n  disambiguate: false\n  finish: false\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.LLM.IsDisambiguateEnabled() {
		t.Error("expected Disambiguate=false when explicitly set")
	}
	if cfg.LLM.IsFinishEnabled() {
		t.Error("expected Finish=false when explicitly set")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := "max_kanji_run: 8\nllm:\n  disambiguate_model: opus\n  finish_model: haiku\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MaxKanjiRun != 8 {
		t.Errorf("expected MaxKanjiRun=8, got %d", cfg.MaxKanjiRun)
	}
	if cfg.LLM.DisambiguateModel != "opus" {
		t.Errorf("expected DisambiguateModel=opus, got %q", cfg.LLM.DisambiguateModel)
	}
	if cfg.LLM.FinishModel != "haiku" {
		t.Errorf("expected FinishModel=haiku, got %q", cfg.LLM.FinishModel)
	}
}

func TestLoad_DefaultPath(t *testing.T) {
	// Load with empty path should not error (falls back to default path which likely doesn't exist)
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxKanjiRun != 5 {
		t.Errorf("expected default config, got MaxKanjiRun=%d", cfg.MaxKanjiRun)
	}
}

func TestLLMConfig_OverrideBackends(t *testing.T) {
	tests := []struct {
		name                    string
		cfg                     LLMConfig
		backend                 string
		disambiguateBackend     string
		wantBackend             string
		wantDisambiguateBackend string
		wantDisambiguateModel   string
		wantFinishModel         string
	}{
		{
			name:                  "same backend keeps models",
			cfg:                   LLMConfig{Backend: BackendCodex, DisambiguateModel: "d", FinishModel: "f"},
			backend:               BackendCodex,
			wantBackend:           BackendCodex,
			wantDisambiguateModel: "d",
			wantFinishModel:       "f",
		},
		{
			name:        "backend change resets both models",
			cfg:         LLMConfig{Backend: BackendCodex, DisambiguateModel: "d", FinishModel: "f"},
			backend:     BackendClaudeCode,
			wantBackend: BackendClaudeCode,
		},
		{
			name:                    "pinned disambiguate backend keeps its model",
			cfg:                     LLMConfig{Backend: BackendCodex, DisambiguateBackend: BackendTypesafeAI, DisambiguateModel: "jev", FinishModel: "f"},
			backend:                 BackendClaudeCode,
			wantBackend:             BackendClaudeCode,
			wantDisambiguateBackend: BackendTypesafeAI,
			wantDisambiguateModel:   "jev",
		},
		{
			name:                    "disambiguate backend change resets only its model",
			cfg:                     LLMConfig{Backend: BackendCodex, DisambiguateModel: "d", FinishModel: "f"},
			disambiguateBackend:     BackendTypesafeAI,
			wantBackend:             BackendCodex,
			wantDisambiguateBackend: BackendTypesafeAI,
			wantFinishModel:         "f",
		},
		{
			name:                    "both flags keep disambiguate model when its effective backend is unchanged",
			cfg:                     LLMConfig{Backend: BackendCodex, DisambiguateModel: "d", FinishModel: "f"},
			backend:                 BackendClaudeCode,
			disambiguateBackend:     BackendCodex,
			wantBackend:             BackendClaudeCode,
			wantDisambiguateBackend: BackendCodex,
			wantDisambiguateModel:   "d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.cfg
			c.OverrideBackends(tt.backend, tt.disambiguateBackend)
			if c.Backend != tt.wantBackend {
				t.Errorf("Backend = %q, want %q", c.Backend, tt.wantBackend)
			}
			if c.DisambiguateBackend != tt.wantDisambiguateBackend {
				t.Errorf("DisambiguateBackend = %q, want %q", c.DisambiguateBackend, tt.wantDisambiguateBackend)
			}
			if c.DisambiguateModel != tt.wantDisambiguateModel {
				t.Errorf("DisambiguateModel = %q, want %q", c.DisambiguateModel, tt.wantDisambiguateModel)
			}
			if c.FinishModel != tt.wantFinishModel {
				t.Errorf("FinishModel = %q, want %q", c.FinishModel, tt.wantFinishModel)
			}
		})
	}
}

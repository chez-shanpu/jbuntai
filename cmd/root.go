package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/jbuntai/internal/config"
	"github.com/chez-shanpu/jbuntai/internal/llm"
	_ "github.com/chez-shanpu/jbuntai/internal/llm/claudecode" // Register claudecode backend
	_ "github.com/chez-shanpu/jbuntai/internal/llm/codex"      // Register codex backend
	_ "github.com/chez-shanpu/jbuntai/internal/llm/typesafeai" // Register typesafeai disambiguate backend
	"github.com/chez-shanpu/jbuntai/internal/pipeline"
	"github.com/chez-shanpu/jbuntai/internal/postprocess"
	"github.com/chez-shanpu/jbuntai/internal/preprocess"
)

var (
	version = "dev"
	commit  = "none"
)

var (
	flagLLM                 bool
	flagStats               bool
	flagDebug               bool
	flagOutput              string
	flagConfig              string
	flagBackend             string
	flagDisambiguateBackend string
)

var rootCmd = &cobra.Command{
	Use:   "jbuntai [file...]",
	Short: "Convert Japanese text to information style",
	Long:  "jbuntai converts Japanese text to information style (情報文体) by removing unnecessary particles, endings, and replacing particles with symbols.",
	Args:  cobra.ArbitraryArgs,
	RunE:  run,
}

func init() {
	rootCmd.Version = fmt.Sprintf("%s (commit: %s)", version, commit)
	rootCmd.Flags().BoolVar(&flagLLM, "llm", true, "Enable LLM-assisted conversion")
	rootCmd.Flags().BoolVar(&flagStats, "stats", false, "Print compression statistics to stderr")
	rootCmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Write output to file (default: stdout)")
	rootCmd.Flags().StringVar(&flagConfig, "config", "", "Path to config file (default: ~/.config/jbuntai/config.yaml)")
	rootCmd.Flags().BoolVar(&flagDebug, "debug", false, "Print debug logs with timestamps to stderr")
	rootCmd.Flags().StringVar(&flagBackend, "backend", "", fmt.Sprintf("LLM backend: %s (overrides config; disambiguation still uses disambiguate_backend if set)", strings.Join(llm.FinisherBackends(), ", ")))
	rootCmd.Flags().StringVar(&flagDisambiguateBackend, "disambiguate-backend", "", fmt.Sprintf("LLM backend for disambiguation only: %s (overrides config and --backend)", strings.Join(llm.DisambiguatorBackends(), ", ")))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Create debug logger
	level := slog.LevelInfo
	if flagDebug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	logger.Debug("loading config")
	cfg, err := config.Load(flagConfig)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Override backends from CLI flags
	if flagBackend != "" && !slices.Contains(llm.FinisherBackends(), flagBackend) {
		return fmt.Errorf("unknown backend %q: must be one of %s", flagBackend, strings.Join(llm.FinisherBackends(), ", "))
	}
	if flagDisambiguateBackend != "" && !slices.Contains(llm.DisambiguatorBackends(), flagDisambiguateBackend) {
		return fmt.Errorf("unknown disambiguate backend %q: must be one of %s", flagDisambiguateBackend, strings.Join(llm.DisambiguatorBackends(), ", "))
	}
	cfg.LLM.OverrideBackends(flagBackend, flagDisambiguateBackend)

	logger.Debug("reading input")
	input, err := readFromArgs(args)
	if err != nil {
		return err
	}

	// Preprocess
	logger.Debug("starting preprocess")
	preprocessed, blocks := preprocess.Do(input)

	// Run pipeline
	logger.Debug("starting pipeline")
	p, err := pipeline.New(cfg, flagLLM, pipeline.WithLogger(logger))
	if err != nil {
		return fmt.Errorf("pipeline error: %w", err)
	}
	result := p.Run(cmd.Context(), preprocessed)

	// Postprocess
	logger.Debug("starting postprocess")
	output := postprocess.Do(result, blocks)

	if flagStats {
		printStats(input, output)
	}

	logger.Debug("starting output")
	return writeOutput(output)
}

func printStats(input, output string) {
	origLen := len([]rune(input))
	outLen := len([]rune(output))
	if origLen == 0 {
		fmt.Fprintf(os.Stderr, "Original: 0 chars, Converted: %d chars, Ratio: N/A\n", outLen)
		return
	}
	ratio := float64(outLen) / float64(origLen) * 100
	fmt.Fprintf(os.Stderr, "Original: %d chars, Converted: %d chars, Ratio: %.1f%%\n", origLen, outLen, ratio)
}

func writeOutput(data string) error {
	if flagOutput != "" {
		return writeOutputFile(flagOutput, data)
	}

	return writeOutputStdout(data)
}

func writeOutputFile(path string, data string) error {
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	return nil
}

func writeOutputStdout(data string) error {
	writer := bufio.NewWriter(os.Stdout)
	if _, err := writer.WriteString(data); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return writer.Flush()
}

func readFromArgs(args []string) (string, error) {
	if len(args) == 0 {
		return readStdin()
	}
	return readFiles(args)
}

func readStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read stdin: %w", err)
	}
	return string(data), nil
}

func readFiles(paths []string) (string, error) {
	var parts []string
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", p, err)
		}
		parts = append(parts, string(data))
	}
	return strings.Join(parts, "\n"), nil
}

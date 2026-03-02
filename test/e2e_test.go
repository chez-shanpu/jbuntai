//go:build e2e

package test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build binary
	root := filepath.Join("..")
	binaryPath = filepath.Join(root, "bin", "jbuntai")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("failed to build binary: " + err.Error() + "\n" + string(out))
	}

	os.Exit(m.Run())
}

func TestStdinConversion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple desu/masu removal",
			input: "分析します。\n",
			want:  "分析し。\n",
		},
		{
			name:  "particle deletion",
			input: "システムは動作します。\n",
			want:  "システム動作し。\n",
		},
		{
			name:  "parallel listing",
			input: "犬と猫\n",
			want:  "犬・猫\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runBinary(t, tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFileConversion(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		expectedFile string
	}{
		{
			name:         "business text",
			inputFile:    "testdata/business.txt",
			expectedFile: "testdata/business_expected.txt",
		},
		{
			name:         "technical text",
			inputFile:    "testdata/technical.txt",
			expectedFile: "testdata/technical_expected.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected, err := os.ReadFile(tt.expectedFile)
			if err != nil {
				t.Fatalf("failed to read expected file: %v", err)
			}

			cmd := exec.Command(binaryPath, "--llm=false", tt.inputFile)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("binary failed: %v\n%s", err, string(out))
			}

			got := string(out)
			want := string(expected)
			if got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestStatsFlag(t *testing.T) {
	cmd := exec.Command(binaryPath, "--llm=false", "--stats")
	cmd.Stdin = strings.NewReader("分析します。\n")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("binary failed: %v", err)
	}

	// Check that stats were printed to stderr
	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "Original:") || !strings.Contains(stderrStr, "Ratio:") {
		t.Errorf("expected stats in stderr, got: %q", stderrStr)
	}

	// Check that output is still correct on stdout
	if !strings.Contains(stdout.String(), "分析し。") {
		t.Errorf("expected converted text on stdout, got: %q", stdout.String())
	}
}

func TestOutputFlag(t *testing.T) {
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.txt")

	cmd := exec.Command(binaryPath, "--llm=false", "--output", outFile)
	cmd.Stdin = strings.NewReader("分析します。\n")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("binary failed: %v\n%s", err, string(out))
	}

	// Verify the output file was created with correct content
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	got := string(data)
	want := "分析し。\n"
	if got != want {
		t.Errorf("output file content: got %q, want %q", got, want)
	}
}

func TestVersionCommand(t *testing.T) {
	cmd := exec.Command(binaryPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %v\n%s", err, string(out))
	}

	if !strings.Contains(string(out), "jbuntai version") {
		t.Errorf("expected version output, got: %q", string(out))
	}
}

func runBinary(t *testing.T, input string) string {
	t.Helper()
	cmd := exec.Command(binaryPath, "--llm=false")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("binary failed: %v\n%s", err, string(out))
	}

	return string(out)
}

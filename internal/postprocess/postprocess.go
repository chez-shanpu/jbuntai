package postprocess

import (
	"strings"
	"unicode"

	"github.com/chez-shanpu/jbuntai/internal/preprocess"
)

// Do performs postprocessing on the converted text.
// Restores code blocks and adjusts spacing.
func Do(text string, blocks []preprocess.CodeBlock) string {
	// Adjust spacing and clean up spaces before restoring code blocks
	// to preserve intentional consecutive spaces inside code blocks.
	text = adjustSpacing(text)
	text = cleanSpaces(text)

	text = restoreCodeBlocks(text, blocks)

	return text
}

// restoreCodeBlocks replaces placeholders in text with original code block content.
func restoreCodeBlocks(text string, blocks []preprocess.CodeBlock) string {
	for _, b := range blocks {
		text = strings.ReplaceAll(text, b.Placeholder, b.Content)
	}
	return text
}

// adjustSpacing adds a space between CJK and Latin characters where needed.
func adjustSpacing(text string) string {
	runes := []rune(text)
	if len(runes) < 2 {
		return text
	}

	var sb strings.Builder
	sb.Grow(len(text) + len(text)/10)

	for i, r := range runes {
		sb.WriteRune(r)
		if i+1 < len(runes) {
			next := runes[i+1]
			if (isCJK(r) && isLatin(next)) || (isLatin(r) && isCJK(next)) {
				// Only add space if there isn't one already
				if r != ' ' && next != ' ' {
					sb.WriteRune(' ')
				}
			}
		}
	}

	return sb.String()
}

// cleanSpaces replaces multiple consecutive spaces with a single space.
func cleanSpaces(text string) string {
	var sb strings.Builder
	prevSpace := false
	for _, r := range text {
		if r == ' ' {
			if !prevSpace {
				sb.WriteRune(r)
			}
			prevSpace = true
		} else {
			sb.WriteRune(r)
			prevSpace = false
		}
	}
	return sb.String()
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r)
}

func isLatin(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

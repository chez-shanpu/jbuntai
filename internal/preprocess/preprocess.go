package preprocess

import (
	"fmt"
	"regexp"
)

// CodeBlock represents an extracted code block with its placeholder.
type CodeBlock struct {
	Placeholder string
	Content     string
}

// PlaceholderPrefix is the marker used in code block placeholders.
// Referenced by the tokenizer to skip placeholder lines during tokenization.
const PlaceholderPrefix = "%%CODEBLOCK_"

var (
	codeBlockRe  = regexp.MustCompile("(?s)```.*?```")
	inlineCodeRe = regexp.MustCompile("`[^`\\n]+`")
	emojiRe      = regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{1F1E0}-\x{1F1FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}]`)
)

// Do performs preprocessing on the input text.
// Returns the processed text and extracted code blocks for later restoration.
func Do(text string) (string, []CodeBlock) {
	var blocks []CodeBlock

	counter := 0
	text, counter, blocks = extractBlocks(codeBlockRe, text, counter, blocks)
	text, _, blocks = extractBlocks(inlineCodeRe, text, counter, blocks)

	// Remove emoji
	text = emojiRe.ReplaceAllString(text, "")

	// Normalize full-width ASCII to half-width
	text = normalizeFullWidth(text)

	return text, blocks
}

// extractBlocks replaces all matches of re in text with placeholders,
// appending each matched block to blocks and incrementing counter.
func extractBlocks(re *regexp.Regexp, text string, counter int, blocks []CodeBlock) (string, int, []CodeBlock) {
	text = re.ReplaceAllStringFunc(text, func(match string) string {
		placeholder := placeholderFor(counter)
		blocks = append(blocks, CodeBlock{
			Placeholder: placeholder,
			Content:     match,
		})
		counter++
		return placeholder
	})
	return text, counter, blocks
}

func placeholderFor(i int) string {
	return fmt.Sprintf("%%%%%s%d%%%%", PlaceholderPrefix, i)
}

// normalizeFullWidth converts full-width ASCII characters to half-width.
func normalizeFullWidth(text string) string {
	runes := []rune(text)
	for i, r := range runes {
		// Full-width ASCII: U+FF01 to U+FF5E → U+0021 to U+007E
		if r >= 0xFF01 && r <= 0xFF5E {
			runes[i] = r - 0xFF01 + 0x0021
		}
		// Full-width space
		if r == 0x3000 {
			runes[i] = ' '
		}
	}
	return string(runes)
}

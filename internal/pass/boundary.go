package pass

import (
	"unicode"

	"github.com/chez-shanpu/jbuntai/internal/config"
)

// BoundaryPass checks for long kanji runs and restores deleted particles as boundaries.
type BoundaryPass struct {
	maxKanjiRun int
}

func NewBoundaryPass(cfg *config.Config) *BoundaryPass {
	maxRun := cfg.MaxKanjiRun
	if maxRun <= 0 {
		maxRun = 5
	}
	return &BoundaryPass{maxKanjiRun: maxRun}
}

func (p *BoundaryPass) Name() string {
	return "boundary"
}

func (p *BoundaryPass) Apply(tokens []Token) []Token {
	// Check for long kanji runs and restore deleted particles if needed
	for {
		startIdx, endIdx := p.findLongKanjiRun(tokens)
		if startIdx < 0 {
			break
		}

		// Try to restore a deleted particle within the run
		restored := p.restoreParticle(tokens, startIdx, endIdx)
		if !restored {
			break
		}
	}

	return tokens
}

// findLongKanjiRun finds the first kanji run exceeding maxKanjiRun.
// Returns the token index range [startIdx, endIdx) covering the run.
func (p *BoundaryPass) findLongKanjiRun(tokens []Token) (startIdx int, endIdx int) {
	kanjiCount := 0
	runStart := -1
	runEnd := -1

	for i, t := range tokens {
		if t.Deleted() {
			continue
		}

		for _, r := range t.Result() {
			if isKanji(r) {
				if kanjiCount == 0 {
					runStart = i
				}
				runEnd = i + 1
				kanjiCount++
			} else {
				if kanjiCount > p.maxKanjiRun {
					return runStart, runEnd
				}
				kanjiCount = 0
				runStart = -1
			}
		}
	}

	if kanjiCount > p.maxKanjiRun {
		return runStart, runEnd
	}
	return -1, -1
}

// restoreParticle tries to restore a deleted の/な/に within the token range.
func (p *BoundaryPass) restoreParticle(tokens []Token, start, end int) bool {
	// Look for deleted particles that can serve as boundaries
	for i := start; i < end && i < len(tokens); i++ {
		t := &tokens[i]
		if !t.Deleted() {
			continue
		}
		if t.IsParticle() {
			switch t.Surface() {
			case "の", "な", "に":
				t.SetDeleted(false)
				t.SetResult(t.Surface())
				return true
			}
		}
	}
	return false
}

// isKanji checks if a rune is a CJK unified ideograph.
func isKanji(r rune) bool {
	return unicode.Is(unicode.Han, r)
}

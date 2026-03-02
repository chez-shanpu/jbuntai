package pass

// EndingPass removes sentence-ending auxiliary verbs and particles.
type EndingPass struct{}

func NewEndingPass() *EndingPass {
	return &EndingPass{}
}

func (p *EndingPass) Name() string {
	return "ending"
}

func (p *EndingPass) Apply(tokens []Token) []Token {
	n := len(tokens)
	if n == 0 {
		return tokens
	}

	// Find sentence boundaries (句点) and process each sentence segment
	segments := p.findSentenceSegments(tokens)
	for _, seg := range segments {
		p.removeEndings(tokens, seg.start, seg.end)
	}

	return tokens
}

// sentenceSegment represents a contiguous range of tokens forming a sentence.
type sentenceSegment struct {
	start, end int // [start, end) token index range
}

// findSentenceSegments splits tokens into sentence segments delimited by 。.
func (p *EndingPass) findSentenceSegments(tokens []Token) []sentenceSegment {
	var segments []sentenceSegment
	start := 0
	for i, t := range tokens {
		if t.Surface() == "。" && t.IsSymbol() {
			segments = append(segments, sentenceSegment{start: start, end: i + 1})
			start = i + 1
		}
	}
	// Trailing segment without 。
	if start < len(tokens) {
		segments = append(segments, sentenceSegment{start: start, end: len(tokens)})
	}
	return segments
}

// removeEndings processes sentence-final forms within [start, end).
func (p *EndingPass) removeEndings(tokens []Token, start, end int) {
	for i := end - 1; i >= start; i-- {
		t := &tokens[i]

		if t.Deleted() {
			continue
		}
		if t.IsSymbol() {
			continue
		}

		// Handle sentence-ending auxiliary verbs
		if t.IsAuxVerb() {
			base := t.BaseForm()
			if base == "" {
				base = t.Surface()
			}
			switch base {
			case "です", "ます", "だ", "た":
				if isSentenceEndInRange(tokens, i, end) {
					t.SetDeleted(true)
					continue
				}
			}
		}

		// Handle ました/でした patterns (ます+た, です+た)
		if t.IsAuxVerb() && t.Surface() == "た" {
			if i > start {
				prev := &tokens[i-1]
				if prev.IsAuxVerb() {
					prevBase := prev.BaseForm()
					if prevBase == "" {
						prevBase = prev.Surface()
					}
					if prevBase == "ます" || prevBase == "です" {
						if isSentenceEndInRange(tokens, i, end) {
							t.SetDeleted(true)
							prev.SetDeleted(true)
							continue
						}
					}
				}
			}
		}

		// Handle である pattern (で+ある)
		if t.IsVerb() && t.BaseForm() == "ある" {
			if i > start {
				prev := &tokens[i-1]
				if prev.IsAuxVerb() && prev.Surface() == "で" {
					if isSentenceEndInRange(tokens, i, end) {
						t.SetDeleted(true)
						prev.SetDeleted(true)
						continue
					}
				}
			}
		}

		// Handle sentence-ending particles (終助詞)
		if t.IsParticle() && t.POSSub1() == "終助詞" {
			switch t.Surface() {
			case "よ", "ね", "な", "さ", "かな":
				t.SetDeleted(true)
				continue
			}
		}

		// Stop at first non-deletable token from the end of this segment
		break
	}
}

// isSentenceEndInRange checks if position i is at or near the end of a sentence within [pos, end).
func isSentenceEndInRange(tokens []Token, pos int, end int) bool {
	for j := pos + 1; j < end; j++ {
		if tokens[j].Deleted() {
			continue
		}
		if tokens[j].IsSymbol() {
			continue
		}
		return false
	}
	return true
}

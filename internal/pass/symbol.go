package pass

import (
	"github.com/chez-shanpu/jbuntai/internal/config"
)

// locationSuffixes are noun suffixes that indicate a place.
var locationSuffixes = map[string]bool{
	"室": true, "場": true, "館": true, "所": true,
	"駅": true, "店": true, "内": true, "上": true,
	"中": true, "先": true,
}

// quotationVerbs are verbs that follow quotation particle と.
var quotationVerbs = map[string]bool{
	"いう": true, "言う": true, "思う": true, "考える": true,
}

// SymbolPass converts particles to symbols.
type SymbolPass struct {
	cfg             *config.Config
	llmOn           bool
	disambiguations map[int]string // token index → answer (per-line, set by Pipeline)
}

func NewSymbolPass(cfg *config.Config, llmOn bool) *SymbolPass {
	return &SymbolPass{cfg: cfg, llmOn: llmOn}
}

// SetDisambiguations sets the disambiguation results for the current line.
func (p *SymbolPass) SetDisambiguations(d map[int]string) {
	p.disambiguations = d
}

// getDisambiguation returns the disambiguation answer for the token at index i, or empty string if none.
func (p *SymbolPass) getDisambiguation(i int) string {
	if p.disambiguations == nil {
		return ""
	}
	return p.disambiguations[i]
}

func (p *SymbolPass) Name() string {
	return "symbol"
}

func (p *SymbolPass) Apply(tokens []Token) []Token {
	n := len(tokens)
	for i := range n {
		t := &tokens[i]
		if t.Deleted() || !t.IsParticle() {
			continue
		}

		switch {
		case t.Surface() == "で" && t.POSSub1() == "格助詞":
			// で (格助詞): use disambiguation if available, else heuristic
			answer := p.getDisambiguation(i)
			if answer == "location" || (answer == "" && p.isDeLocation(tokens, i)) {
				setPrefixOnNounPhrase(tokens, i, "@")
				t.SetDeleted(true)
			}

		case t.Surface() == "に" && t.POSSub1() == "格助詞":
			// に (格助詞): use disambiguation if available, else heuristic
			answer := p.getDisambiguation(i)
			if answer == "direction" || (answer == "" && isDirectionTarget(tokens, i)) {
				setPrefixOnNounPhrase(tokens, i, ">")
				t.SetDeleted(true)
			}

		case t.Surface() == "へ" && t.POSSub1() == "格助詞":
			// へ (格助詞) → > prefix
			setPrefixOnNounPhrase(tokens, i, ">")
			t.SetDeleted(true)

		case t.Surface() == "から" && hasFollowingMade(tokens, i):
			// から...まで → 〜 (range)
			t.SetResult("〜")

		case t.Surface() == "まで":
			// まで in range context → 〜
			if hasPrecedingKara(tokens, i) {
				t.SetResult("〜")
			}

		case t.Surface() == "と" && t.POSSub1() == "格助詞":
			// と (格助詞): use disambiguation if available, else heuristic
			answer := p.getDisambiguation(i)
			if answer == "quotation" || (answer == "" && p.isQuotationTo(tokens, i)) {
				t.SetResult(":")
			} else if answer == "parallel" || (answer == "" && p.isParallelToYa(tokens, i)) {
				t.SetResult("・")
			}

		case (t.Surface() == "や" || (t.Surface() == "と" && t.POSSub1() == "並立助詞")) && p.isParallelToYa(tokens, i):
			// や/と (並立助詞) between nouns → ・
			t.SetResult("・")

		case t.Surface() == "ので" || (t.Surface() == "ため" && t.IsParticle()):
			// ので/ため (reason) → ∵ prefix
			setPrefixOnNounPhrase(tokens, i, "∵")
			t.SetDeleted(true)

		case isConjunctiveTeDe(t):
			// て/で (接続助詞) → ,
			t.SetResult(",")
		}
	}

	return tokens
}

// isDeLocation checks if で at position i follows a location noun.
func (p *SymbolPass) isDeLocation(tokens []Token, i int) bool {
	t := &tokens[i]
	if t.Surface() != "で" || t.POSSub1() != "格助詞" {
		return false
	}

	// Check preceding token for location
	prev := FindPrevNonDeleted(tokens, i)
	if prev < 0 {
		return false
	}

	pt := &tokens[prev]

	// Check for proper noun (地域)
	if pt.IsNoun() && pt.POSSub1() == "固有名詞" && pt.POSSub2() == "地域" {
		return true
	}

	// Check for location suffix
	if pt.IsNoun() && pt.POSSub1() == "接尾" {
		if locationSuffixes[pt.Surface()] {
			return true
		}
	}

	// Check surface directly for location suffixes
	surface := pt.Surface()
	runes := []rune(surface)
	if len(runes) > 0 {
		lastRune := runes[len(runes)-1]
		if locationSuffixes[string(lastRune)] {
			return true
		}
	}

	return false
}

// isDirectionTarget checks if に at position i is used for direction/target.
func isDirectionTarget(tokens []Token, i int) bool {
	// Simple heuristic: に followed by a verb suggests direction
	next := FindNextNonDeleted(tokens, i)
	if next >= 0 && tokens[next].IsVerb() {
		return true
	}
	return false
}

// hasFollowingMade checks if there is a まで after position i.
func hasFollowingMade(tokens []Token, i int) bool {
	for j := i + 1; j < len(tokens); j++ {
		if tokens[j].Deleted() {
			continue
		}
		if tokens[j].Surface() == "まで" {
			return true
		}
		// Stop at sentence boundaries
		if tokens[j].IsSymbol() && (tokens[j].Surface() == "。" || tokens[j].Surface() == "、") {
			return false
		}
	}
	return false
}

// hasPrecedingKara checks if there is a から before position i.
func hasPrecedingKara(tokens []Token, i int) bool {
	for j := i - 1; j >= 0; j-- {
		if tokens[j].Deleted() {
			continue
		}
		if tokens[j].Surface() == "から" {
			return true
		}
		if tokens[j].IsSymbol() && (tokens[j].Surface() == "。" || tokens[j].Surface() == "、") {
			return false
		}
	}
	return false
}

// isQuotationTo checks if と at position i is followed by a quotation verb.
func (p *SymbolPass) isQuotationTo(tokens []Token, i int) bool {
	t := &tokens[i]
	if t.Surface() != "と" || t.POSSub1() != "格助詞" {
		return false
	}

	next := FindNextNonDeleted(tokens, i)
	if next < 0 {
		return false
	}

	base := tokens[next].BaseForm()
	if base == "" {
		base = tokens[next].Surface()
	}
	return quotationVerbs[base]
}

// isParallelToYa checks if と/や at position i is between nouns (parallel listing).
func (p *SymbolPass) isParallelToYa(tokens []Token, i int) bool {
	t := &tokens[i]
	if t.Surface() != "と" && t.Surface() != "や" {
		return false
	}
	if t.POSSub1() != "並立助詞" && t.POSSub1() != "格助詞" {
		return false
	}

	prev := FindPrevNonDeleted(tokens, i)
	next := FindNextNonDeleted(tokens, i)

	if prev < 0 || next < 0 {
		return false
	}

	return tokens[prev].IsNoun() && tokens[next].IsNoun()
}

// isConjunctiveTeDe checks if the token is a conjunctive て/で.
func isConjunctiveTeDe(t *Token) bool {
	if t.POSSub1() != "接続助詞" {
		return false
	}
	return t.Surface() == "て" || t.Surface() == "で"
}

// setPrefixOnNounPhrase finds the start of the noun phrase before position i
// and sets the prefix on it.
func setPrefixOnNounPhrase(tokens []Token, particleIdx int, prefix string) {
	// Walk backward from the particle to find the noun phrase start
	start := findNounPhraseStart(tokens, particleIdx)
	if start >= 0 {
		tokens[start].SetPrefix(prefix)
	}
}

// findNounPhraseStart finds the beginning of the noun phrase ending at particleIdx.
func findNounPhraseStart(tokens []Token, particleIdx int) int {
	pos := FindPrevNonDeleted(tokens, particleIdx)
	if pos < 0 {
		return -1
	}

	// Walk backward through nouns and particles that form a compound
	start := pos
	for {
		prev := FindPrevNonDeleted(tokens, start)
		if prev < 0 {
			break
		}
		pt := &tokens[prev]
		if pt.IsNoun() || (pt.IsParticle() && pt.Surface() == "の") {
			start = prev
		} else {
			break
		}
	}

	return start
}

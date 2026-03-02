package pass

// DeletionPass removes unnecessary particles.
type DeletionPass struct{}

func NewDeletionPass() *DeletionPass {
	return &DeletionPass{}
}

func (p *DeletionPass) Name() string {
	return "deletion"
}

func (p *DeletionPass) Apply(tokens []Token) []Token {
	n := len(tokens)
	for i := range n {
		t := &tokens[i]
		if t.Deleted() {
			continue
		}

		// Handle non-independent nouns (こと/もの)
		if isNonIndependentNoun(t) {
			t.SetDeleted(true)
			continue
		}

		if !t.IsParticle() {
			continue
		}

		switch {
		case t.Surface() == "は" && t.POSSub1() == "係助詞":
			// は (係助詞) → delete
			t.SetDeleted(true)

		case t.Surface() == "が" && t.POSSub1() == "格助詞":
			// が (格助詞, subject) → delete, but keep adversative が
			if !isAdversativeGa(tokens, i) {
				t.SetDeleted(true)
			}

		case t.Surface() == "を" && t.POSSub1() == "格助詞":
			// を (格助詞) → delete when followed by sino-japanese verb
			if isFollowedBySinoVerb(tokens, i) {
				t.SetDeleted(true)
			}
		}
	}

	return tokens
}

// isAdversativeGa checks if が at position i is adversative (逆接).
// Heuristic: if followed by 、 (comma), it's likely adversative.
func isAdversativeGa(tokens []Token, i int) bool {
	next := FindNextNonDeleted(tokens, i)
	if next < 0 {
		return false
	}
	return tokens[next].Surface() == "、"
}

// isFollowedBySinoVerb checks if を is followed by a sino-japanese verb (サ変接続 noun + する).
func isFollowedBySinoVerb(tokens []Token, i int) bool {
	next := FindNextNonDeleted(tokens, i)
	if next < 0 {
		return false
	}

	nt := &tokens[next]

	// Direct verb following を
	if nt.IsVerb() {
		return true
	}

	// サ変接続 noun (e.g., 検討を行う)
	if nt.IsNoun() && nt.POSSub1() == "サ変接続" {
		return true
	}

	return false
}

// isNonIndependentNoun checks if the token is a non-independent noun (非自立名詞).
func isNonIndependentNoun(t *Token) bool {
	if !t.IsNoun() || t.POSSub1() != "非自立" {
		return false
	}
	return t.Surface() == "こと" || t.Surface() == "もの"
}

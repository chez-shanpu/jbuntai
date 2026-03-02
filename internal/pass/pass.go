package pass

// Token represents a morphological token with conversion state.
type Token struct {
	surface  string   // Original surface form (immutable)
	pos      []string // Part-of-speech tags (immutable)
	baseForm string   // Base/dictionary form (immutable)
	reading  string   // Reading in katakana (immutable)
	result   string   // Conversion result (initially = surface)
	deleted  bool     // Deletion flag
	prefix   string   // Prefix symbol (e.g., @, >, ∵)
}

// NewToken creates a Token. Result is initially set to surface.
func NewToken(surface string, pos []string, baseForm, reading string) Token {
	return Token{
		surface:  surface,
		pos:      pos,
		baseForm: baseForm,
		reading:  reading,
		result:   surface,
	}
}

func (t Token) Surface() string  { return t.surface }
func (t Token) POS() []string    { return t.pos }
func (t Token) BaseForm() string { return t.baseForm }
func (t Token) Reading() string  { return t.reading }
func (t Token) Result() string   { return t.result }
func (t Token) Deleted() bool    { return t.deleted }
func (t Token) Prefix() string   { return t.prefix }

func (t *Token) SetResult(s string) { t.result = s }
func (t *Token) SetDeleted(d bool)  { t.deleted = d }
func (t *Token) SetPrefix(s string) { t.prefix = s }

// Pass defines a transformation pass over tokens.
type Pass interface {
	Name() string
	Apply(tokens []Token) []Token
}

func (t Token) POSClass() string {
	if len(t.pos) > 0 {
		return t.pos[0]
	}
	return ""
}

func (t Token) POSSub1() string {
	if len(t.pos) > 1 {
		return t.pos[1]
	}
	return ""
}

func (t Token) POSSub2() string {
	if len(t.pos) > 2 {
		return t.pos[2]
	}
	return ""
}

func (t Token) POSSub3() string {
	if len(t.pos) > 3 {
		return t.pos[3]
	}
	return ""
}

// IsParticle returns true if the token is a particle (助詞).
func (t Token) IsParticle() bool {
	return t.POSClass() == "助詞"
}

// IsAuxVerb returns true if the token is an auxiliary verb (助動詞).
func (t Token) IsAuxVerb() bool {
	return t.POSClass() == "助動詞"
}

// IsNoun returns true if the token is a noun (名詞).
func (t Token) IsNoun() bool {
	return t.POSClass() == "名詞"
}

// IsVerb returns true if the token is a verb (動詞).
func (t Token) IsVerb() bool {
	return t.POSClass() == "動詞"
}

// IsSymbol returns true if the token is a symbol/punctuation (記号).
func (t Token) IsSymbol() bool {
	return t.POSClass() == "記号"
}

// FindPrevNonDeleted finds the previous non-deleted token before position i.
func FindPrevNonDeleted(tokens []Token, i int) int {
	for j := i - 1; j >= 0; j-- {
		if !tokens[j].Deleted() {
			return j
		}
	}
	return -1
}

// FindNextNonDeleted finds the next non-deleted token after position i.
func FindNextNonDeleted(tokens []Token, i int) int {
	for j := i + 1; j < len(tokens); j++ {
		if !tokens[j].Deleted() {
			return j
		}
	}
	return -1
}

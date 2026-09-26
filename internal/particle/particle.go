// Package particle defines the usage labels of ambiguous case particles
// (で, に, と). It is the single source of truth shared by the pipeline
// (which asks for disambiguation), the symbol pass (which consumes the
// answers), and LLM backends (which describe the labels to the model).
package particle

// Usage labels returned by disambiguation.
const (
	LabelLocation  = "location"
	LabelMeans     = "means"
	LabelDirection = "direction"
	LabelTime      = "time"
	LabelQuotation = "quotation"
	LabelParallel  = "parallel"
	LabelCompanion = "companion"
	LabelOther     = "other"
)

// Choice is a usage label of a particle and what it means.
type Choice struct {
	Label       string
	Description string
}

var caseParticleChoices = map[string][]Choice{
	"で": {
		{LabelLocation, "で marks the location where an action takes place (e.g. 会議室で話す = talk at/in the meeting room)"},
		{LabelMeans, "で marks the means or instrument used to perform an action (e.g. 電車で行く = go by train)"},
		{LabelOther, "で is used in a way that is neither a location nor a means"},
	},
	"に": {
		{LabelDirection, "に marks the direction or target of movement/action (e.g. 東京に行く = go to Tokyo)"},
		{LabelLocation, "に marks the location where something exists or is fixed (e.g. 東京に住む = live in Tokyo)"},
		{LabelTime, "に marks a point in time (e.g. 3時に会う = meet at 3 o'clock)"},
		{LabelOther, "に is used in a way that is not direction, location, or time"},
	},
	"と": {
		{LabelQuotation, "と marks a quotation or reported thought/speech (e.g. 重要だと判断した = judged that it is important)"},
		{LabelParallel, "と lists items side by side, equivalent to \"and\" (e.g. 犬と猫 = dog and cat)"},
		{LabelCompanion, "と marks a companion doing the action together (e.g. 友人と話す = talk with a friend)"},
		{LabelOther, "と is used in a way that is not quotation, parallel, or companion"},
	},
}

// CaseParticleLabels returns the usage labels of the given case particle
// (格助詞) surface, or nil if the particle needs no disambiguation.
func CaseParticleLabels(surface string) []string {
	choices := caseParticleChoices[surface]
	if len(choices) == 0 {
		return nil
	}
	labels := make([]string, len(choices))
	for i, c := range choices {
		labels[i] = c.Label
	}
	return labels
}

// Description returns the description of the given label of a case particle.
func Description(surface, label string) (string, bool) {
	for _, c := range caseParticleChoices[surface] {
		if c.Label == label {
			return c.Description, true
		}
	}
	return "", false
}

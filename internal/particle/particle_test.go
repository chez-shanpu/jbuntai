package particle

import (
	"slices"
	"testing"
)

func TestCaseParticleLabels(t *testing.T) {
	tests := []struct {
		surface string
		want    []string
	}{
		{"で", []string{LabelLocation, LabelMeans, LabelOther}},
		{"に", []string{LabelDirection, LabelLocation, LabelTime, LabelOther}},
		{"と", []string{LabelQuotation, LabelParallel, LabelCompanion, LabelOther}},
		{"が", nil},
	}
	for _, tt := range tests {
		t.Run(tt.surface, func(t *testing.T) {
			if got := CaseParticleLabels(tt.surface); !slices.Equal(got, tt.want) {
				t.Errorf("CaseParticleLabels(%q) = %v, want %v", tt.surface, got, tt.want)
			}
		})
	}
}

func TestDescriptionCoversAllLabels(t *testing.T) {
	for surface := range caseParticleChoices {
		for _, label := range CaseParticleLabels(surface) {
			if d, ok := Description(surface, label); !ok || d == "" {
				t.Errorf("missing description for %s/%s", surface, label)
			}
		}
	}
	if _, ok := Description("に", LabelMeans); ok {
		t.Errorf("Description(に, means) should not exist")
	}
}

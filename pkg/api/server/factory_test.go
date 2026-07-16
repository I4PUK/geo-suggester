package server

import (
	"testing"

	"golang.org/x/text/language"
	coremodels "github.com/example/geo/suggest-core/models"
	"github.com/example/golang/locale"
)

// Found that using pointers gives +7% to speed for SuggestionToProto
// BenchmarkSuggestionToProto-12_value            3839838              3162 ns/op             968 B/op         17 allocs/op
// BenchmarkSuggestionToProto-12_pointer          3889158              2946 ns/op             968 B/op         17 allocs/op
func BenchmarkSuggestionToProto(b *testing.B) {
	pop := uint(1234)
	suggestion := coremodels.Suggestion{
		GeoID:    1000,
		GeoType:  "some-type",
		IsActive: true,
		Codes: map[string]string{
			"code1": "codeVal1",
			"code2": "codeVal2",
		},
		GeoLocation: &coremodels.Location{
			Lon: 10,
			Lat: 20,
		},
		Population: &pop,
		Name: map[string]string{
			"en_En": "test",
		},
		ParentNames: map[string][]string{
			"parent1": {"val1", "val2"},
		},
		OwnRepresents: map[string][]string{
			"represent1": {"val1", "val2"},
		},
		OwnRepresentsFuzzy: map[string][]string{
			"represent1": {"val1", "val2"},
		},
		Represents: map[string][]string{
			"represent1": {"val1", "val2"},
		},
		RepresentsFuzzy: map[string][]string{
			"represent1": {"val1", "val2"},
		},
		TypeSpecific: coremodels.TypeSpecific{},
		CustomScores: map[string]coremodels.CustomScore{
			"score1": {
				DefaultScore: 4,
				Weight:       2,
			},
		},
		ChildrenBelongedCount: coremodels.CountByGeoType{
			Hotel: 10,
		},
	}

	l, err := locale.New("en_EN", []language.Tag{language.English})
	if err != nil {
		b.Fatalf("error on locale init")
	}
	for i := 0; i < b.N; i++ {
		SuggestionToProto(&suggestion, l)
	}
}

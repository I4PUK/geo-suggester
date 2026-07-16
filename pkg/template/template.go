package template

import (
	"strings"

	"github.com/example/geo/common"
	"github.com/example/geo/suggest-core/fn"

	"github.com/example/geo-suggest/pkg/models"
)

const (
	PrefixTemplate        = "hotels"
	PrefixFuzzyTemplate   = "_fuzzy"
	PrefixContextTemplate = "_context"
)

func GenerateTemplateID(templateName string) string {
	if strings.Contains(templateName, PrefixFuzzyTemplate) {
		return PrefixTemplate + PrefixFuzzyTemplate
	}

	if strings.Contains(templateName, PrefixContextTemplate) {
		return PrefixTemplate + PrefixContextTemplate
	}

	return PrefixTemplate
}

func GetTemplateID(fuzzy bool) string {
	if fuzzy {
		return PrefixTemplate + "_fuzzy"
	}
	return PrefixTemplate
}

func Params(q models.Query) map[string]interface{} {
	ret := make(map[string]interface{})
	if q.Query != "" {
		words, prefix := fn.PrefixAndWords(fn.CleanUpString(q.Query, common.GeoTypeLocality))
		if words != "" {
			ret["words"] = words
		}
		if prefix != "" {
			ret["prefix"] = prefix
		}
	}
	if q.Location != nil {
		ret["lat"] = q.Location.Lat
		ret["lon"] = q.Location.Lon
	}
	ret["locale"] = fn.MapRequestLocale(q.Locale)
	ret["size"] = q.Limit
	return ret
}

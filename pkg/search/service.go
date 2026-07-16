package search

import (
	"context"
	"strconv"
	"strings"

	"github.com/fordarian/go-lang-correct"
	"github.com/rs/zerolog"
	geomodels "github.com/example/geo/models"
	coremodels "github.com/example/geo/suggest-core/models"

	"github.com/example/geo-suggest/pkg/models"
	"github.com/example/geo-suggest/pkg/template"
)

type Service struct {
	elastic         Elastic
	logger          *zerolog.Logger
	geo             Geo
	filter          filter
	searchTemplates map[string]string
	topInRegions    []uint64
}

type filter interface {
	FilterSuggestion(ctx context.Context, input []coremodels.Suggestion) []coremodels.Suggestion
}

type Elastic interface {
	RunSearchTemplate(ctx context.Context, id string, params map[string]interface{}) ([]coremodels.Suggestion, error)
}

type Geo interface {
	GetLocalityByGeoIdAndGeoType(ctx context.Context, geoId uint64, geoType string) (geomodels.Locality, error)
	GetAdmRegionNameByLocality(ctx context.Context, locality geomodels.Locality, locale string) (string, error)
	GetAdmDistrictNameByLocality(ctx context.Context, locality geomodels.Locality, locale string) (string, error)
	GetCountryNameByLocality(ctx context.Context, locality geomodels.Locality, locale string) (string, error)
}

func NewService(
	storage Elastic,
	logger *zerolog.Logger,
	geo Geo,
	content filter,
	searchTemplates map[string]string,
	topInRegionsStr string,
) *Service {

	return &Service{
		elastic:         storage,
		logger:          logger,
		geo:             geo,
		filter:          content,
		searchTemplates: searchTemplates,
		topInRegions:    topInRegionsStrToUint(topInRegionsStr),
	}
}

func topInRegionsStrToUint(value string) (out []uint64) {
	for _, geoID := range strings.Split(value, ",") {
		id, err := strconv.Atoi(geoID)
		if err != nil {
			continue
		}
		out = append(out, uint64(id))
	}
	return out
}

func (s *Service) Suggest(ctx context.Context, q models.Query) ([]coremodels.Suggestion, error) {
	ret, err := s.suggest(ctx, q, false)
	if err != nil {
		return nil, err
	}

	if len(ret) != 0 {
		return ret, nil
	}

	return s.suggest(ctx, q, true)
}

func (s *Service) suggest(ctx context.Context, q models.Query, fuzzy bool) ([]coremodels.Suggestion, error) {
	templateID := template.GetTemplateID(fuzzy)
	ret, err := s.elastic.RunSearchTemplate(ctx, templateID, template.Params(q))
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		// punto switcher
		nq := lang.Correct(q.Query)
		if nq != q.Query {
			q.Query = nq
			ret, err = s.elastic.RunSearchTemplate(ctx, templateID, template.Params(q))
			if err != nil {
				return nil, err
			}
		}
	}
	// remove blacklisted objects from suggestion
	suggestion := s.filter.FilterSuggestion(ctx, ret)

	suggestion = s.getSuggestionOptions(ctx, suggestion, q.Locale)
	prioritySuggestion := s.getSuggestionOptions(ctx, s.extractHotelsFromSuggestionContext(ctx, suggestion, q.Locale), q.Locale)
	if len(prioritySuggestion) == 0 {
		return suggestion, nil
	}
	suggestion = s.mixSuggestionWithPriority(suggestion, prioritySuggestion)
	return suggestion, nil
}

func (s *Service) mixSuggestionWithPriority(suggestion []coremodels.Suggestion, priority []coremodels.Suggestion) []coremodels.Suggestion {
	out := make([]coremodels.Suggestion, 0)
	u := make(map[uint64]bool)
	for _, sugg := range suggestion {

		if _, ok := u[sugg.GeoID.Uint64()]; !ok {
			if sugg.GeoType == geomodels.GeoTypeLocality.String() {
				out = append(out, sugg)
				for _, p := range priority {
					if _, ok := u[p.GeoID.Uint64()]; !ok {
						out = append(out, p)
						u[p.GeoID.Uint64()] = true
					}
				}
			}

			if sugg.GeoType == geomodels.GeoTypeHotel.String() {
				out = append(out, sugg)
			}
			u[sugg.GeoID.Uint64()] = true // exclude duplicate
		}
	}
	return out
}

func (s *Service) getSuggestionOptions(ctx context.Context, suggestion []coremodels.Suggestion, locale string) []coremodels.Suggestion {
	for _, v := range suggestion {
		switch v.GeoType {
		case geomodels.GeoTypeCountry.String(),
			geomodels.GeoTypeRegion.String(),
			geomodels.GeoTypeAdmRegion.String():
			continue
		}
		locality, err := s.geo.GetLocalityByGeoIdAndGeoType(ctx, v.GeoID.Uint64(), v.GeoType)
		if err != nil {
			s.logger.Warn().Err(err).Msg("cant get locality")
			continue
		}
		admRegionName, err := s.geo.GetAdmRegionNameByLocality(ctx, locality, locale)
		if err != nil {
			s.logger.Warn().Err(err).Msg("cant get adm region name")
			continue
		}
		v.ParentNames[locale] = []string{admRegionName}
	}
	suggestionWithDuplicateNamesByGeoType := make(map[string]map[string]bool)
	for _, v := range suggestion {
		name := v.Name[locale]
		if _, ok := suggestionWithDuplicateNamesByGeoType[name]; !ok {
			suggestionWithDuplicateNamesByGeoType[name] = make(map[string]bool)
		}
		if _, ok := suggestionWithDuplicateNamesByGeoType[name][v.GeoType]; !ok {
			suggestionWithDuplicateNamesByGeoType[name][v.GeoType] = false
		} else {
			suggestionWithDuplicateNamesByGeoType[name][v.GeoType] = true
		}
	}
	for _, v := range suggestion {
		locality, err := s.geo.GetLocalityByGeoIdAndGeoType(ctx, v.GeoID.Uint64(), v.GeoType)
		if err != nil {
			s.logger.Warn().Err(err).Msg("cant get locality")
			continue
		}
		if suggestionWithDuplicateNamesByGeoType[v.Name[locale]][v.GeoType] {
			admDistrictName, err := s.geo.GetAdmDistrictNameByLocality(ctx, locality, locale)
			if err != nil {
				s.logger.Warn().Err(err).Msg("cant get adm district name")
				continue
			}
			v.ParentNames[locale] = append(v.ParentNames[locale], admDistrictName)
		}
		countryName, err := s.geo.GetCountryNameByLocality(ctx, locality, locale)
		if err != nil {
			s.logger.Warn().Err(err).Msg("cant get country name")
			continue
		}
		if !s.parentContains(v.ParentNames[locale], countryName) {
			v.ParentNames[locale] = append(v.ParentNames[locale], countryName)
		}
	}
	return suggestion
}

func (s *Service) parentContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func (s *Service) extractHotelsFromSuggestionContext(ctx context.Context, suggestion []coremodels.Suggestion, locale string) []coremodels.Suggestion {
	searchInContextGeoID := make([]uint64, 0)
	for _, geoID := range s.topInRegions {
		locality, err := s.geo.GetLocalityByGeoIdAndGeoType(ctx, geoID, geomodels.GeoTypeHotel.String())
		if err != nil {
			s.logger.Warn().Err(err).Msg("can't get locality")
		}

		admRegion, err := s.geo.GetAdmRegionNameByLocality(ctx, locality, locale)
		if err != nil {
			s.logger.Warn().Err(err).Msg("can't get adm region name")
			continue
		}

		for _, v := range suggestion {
			if s.parentContains(v.ParentNames[locale], admRegion) {
				searchInContextGeoID = append(searchInContextGeoID, geoID)
				break
			}
		}
	}

	templateID := template.GenerateTemplateID(template.PrefixContextTemplate)
	prioritySuggest, err := s.elastic.RunSearchTemplate(ctx, templateID, map[string]interface{}{
		"geoids": searchInContextGeoID,
	})

	if err != nil || len(searchInContextGeoID) == 0 {
		return nil
	}

	return prioritySuggest
}

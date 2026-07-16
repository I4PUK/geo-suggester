package http

import (
	"net/http"
	"strconv"

	"github.com/example/geo-suggest/pkg/models"
	"github.com/pkg/errors"
)

func (h *SuggestHandler) requestToQuery(r *http.Request) (models.Query, error) {
	reqQuery := r.URL.Query()
	locale := reqQuery.Get("locale")
	if locale == "" {
		locale = "en_EN"
	}
	query := models.Query{
		Query:  reqQuery.Get("query"),
		Locale: locale,
	}
	if query.Query == "" {
		return models.Query{}, errors.Errorf("query is required")
	}

	if limit := reqQuery.Get("limit"); limit == "" {
		query.Limit = DefaultLimit
	} else {
		intLimit, err := strconv.Atoi(limit)
		if err != nil {
			return models.Query{}, errors.Wrap(err, "imvalid limit value")
		}
		query.Limit = uint(intLimit)
	}

	if latStr := reqQuery.Get("lat"); latStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return models.Query{}, errors.New("malformed location")
		}
		query.Location.Lat = lat
	}

	if lonStr := reqQuery.Get("lon"); lonStr != "" {
		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil {
			return models.Query{}, errors.New("malformed location")
		}
		query.Location.Lon = lon
	}
	return query, nil
}

package http

import (
	"net/http"
	"strconv"

	"github.com/example/golang/log"
	"github.com/example/golang/resources/sentry"
	"github.com/example/geo-suggest/pkg/api/server"
)

const (
	DefaultLimit = 10
)

type SuggestHandler struct {
	service server.SearchService
}

func NewSuggestHandler(service server.SearchService) *SuggestHandler {
	return &SuggestHandler{service: service}
}

func (h *SuggestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	addResponseHeaders(w)
	query, err := h.requestToQuery(req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeErrMessage(w, Error{Message: err.Error()})
		return
	}
	res, err := h.service.Suggest(req.Context(), query)
	if err != nil {
		sentry.ReportError(err)
		w.WriteHeader(http.StatusInternalServerError)
		writeErrMessage(w, Error{Message: "Internal error"})
		return
	}

	var respSuggestions []Suggestion
	for i := range res {
		respSuggestion, err := convertSuggestionResponse(&res[i], query.Locale)
		if err != nil {
			log.Logger.Err(err).
				Str("geo_id", strconv.FormatUint(uint64(res[i].GeoID), 10)).
				Str("locale", query.Locale).
				Send()
			continue
		}
		respSuggestions = append(respSuggestions, respSuggestion)
	}
	w.WriteHeader(http.StatusOK)
	writeOKResponse(w, map[string][]Suggestion{"items": respSuggestions})
}

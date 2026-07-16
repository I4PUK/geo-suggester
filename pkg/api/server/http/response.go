package http

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	coremodels "github.com/example/geo/suggest-core/models"
	"github.com/example/golang/resources/sentry"
)

type Suggestion struct {
	GeoID       uint64        `json:"geoID"`
	Name        string        `json:"name"`
	GeoType     string        `json:"geoType"`
	Parents     []HotelParent `json:"parents"`
	GeoLocation *GeoLocation  `json:"geoLocation"`
	HotelsCount uint64        `json:"hotelsCount"`
}

type GeoLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type HotelParent struct {
	Name string `json:"name"`
}

type Error struct {
	Message string `json:"message"`
}

func convertSuggestionResponse(s *coremodels.Suggestion, locale string) (Suggestion, error) {
	/*
		Per unified geography, the main locale is en_EN.
		Sometimes ru also arrives (e.g. railway station Riminifiera) — partner data artifacts.
		If there is no name for the requested locale, skip such suggestions.
	*/
	if s.Name[locale] == "" {
		return Suggestion{}, errors.New("empty locale name")
	}
	ret := Suggestion{
		GeoID:       uint64(s.GeoID),
		Name:        s.Name[locale],
		GeoType:     s.GeoType,
		HotelsCount: s.ChildrenBelongedCount.Hotel,
	}
	parents := make([]HotelParent, 0, len(s.ParentNames[locale]))
	for _, n := range s.ParentNames[locale] {
		if n != "" {
			parents = append(parents, HotelParent{Name: n})
		}
	}
	ret.Parents = parents

	if s.GeoLocation != nil {
		ret.GeoLocation = &GeoLocation{
			Lat: s.GeoLocation.Lat,
			Lon: s.GeoLocation.Lon,
		}
	}
	return ret, nil
}

func writeErrMessage(w http.ResponseWriter, er Error) {
	if err := json.NewEncoder(w).Encode(map[string]Error{"error": er}); err != nil {
		sentry.ReportError(err)
		if er.Message != "error sending response" {
			w.WriteHeader(http.StatusInternalServerError)
			writeErrMessage(w, Error{Message: "error sending response"})
		}
		return
	}
}

func writeOKResponse(w http.ResponseWriter, body interface{}) {
	if err := json.NewEncoder(w).Encode(body); err != nil {
		sentry.ReportError(err)
		w.WriteHeader(http.StatusInternalServerError)
		writeErrMessage(w, Error{"error sending response"})
		return
	}
}

func addResponseHeaders(w http.ResponseWriter) {
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
}

package server

import (
	"errors"
	"strings"

	commonproto "github.com/example/geo/contracts/common/langs/go"
	"github.com/example/geo/suggest-core/fn"
	coremodels "github.com/example/geo/suggest-core/models"
	"github.com/example/golang/locale"
	proto "github.com/example/hotels-search/contracts/geo-suggest/langs/go"
)

func SuggestionToProto(suggestion *coremodels.Suggestion, l *locale.L) (*proto.Suggestion, error) {
	geoType, ok := commonproto.GeoType_value[strings.ToUpper(suggestion.GeoType)]
	if !ok {
		geoType = int32(commonproto.GeoType_GEO_TYPE_UNKNOWN)
	}
	name := l.ResolveFromMap(suggestion.Name)
	if name == "" {
		return nil, errors.New("empty name")
	}
	return &proto.Suggestion{
		GeoId:                uint64(suggestion.GeoID),
		GeoType:              commonproto.GeoType(geoType),
		Codes:                suggestion.Codes,
		LocalizedName:        name,
		LocalizedParentNames: fn.LocalizedParentNames(*suggestion, l),
		GeoLocation:          LocationToProto(suggestion.GeoLocation),
		HotelsCount:          suggestion.ChildrenBelongedCount.Hotel,
	}, nil
}

func SuggestionsToProto(ss []coremodels.Suggestion, l *locale.L) []*proto.Suggestion {
	ret := make([]*proto.Suggestion, 0, len(ss))
	for i := range ss {
		conv, err := SuggestionToProto(&ss[i], l)
		if err == nil {
			ret = append(ret, conv)
		}
	}
	return ret
}

func LocationFromProto(location *proto.GeoLocation) *coremodels.Location {
	if location == nil {
		return nil
	}
	return &coremodels.Location{
		Lon: location.Lon,
		Lat: location.Lat,
	}
}

func LocationToProto(location *coremodels.Location) *proto.GeoLocation {
	if location == nil {
		return nil
	}
	return &proto.GeoLocation{
		Lon: location.Lon,
		Lat: location.Lat,
	}
}

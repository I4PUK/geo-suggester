package filter

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	coremodels "github.com/example/geo/suggest-core/models"

	"github.com/pkg/errors"
	"github.com/example/hotels-modules/cache"
)

const (
	blacklistKey = "htl_blacklist"
	blacklistTTL = time.Hour
	hotelGeoType = "hotel"
)

type filter struct {
	logger        *zerolog.Logger
	cache         cache.Cache
	contentClient contentClient
}

type contentClient interface {
	GetBlackListedHotels(ctx context.Context) ([]uint64, error)
}

func NewFilter(logger *zerolog.Logger, cl contentClient, c cache.Cache) *filter {
	return &filter{
		logger:        logger,
		contentClient: cl,
		cache:         c,
	}
}

func (s *filter) FilterSuggestion(ctx context.Context, input []coremodels.Suggestion) []coremodels.Suggestion {
	result := make([]coremodels.Suggestion, 0, len(input))
	blacklist, err := s.GetAllBlackListHotels(ctx)
	if err != nil {
		// error
		return input
	}
	// post-search filtration
	for i, v := range input {
		// we blacklist only hotels right now
		if _, ok := blacklist[v.GeoID.Uint64()]; ok && v.GeoType == hotelGeoType {
			continue
		}
		result = append(result, input[i])
	}
	return result
}

func (s *filter) GetAllBlackListHotels(ctx context.Context) (map[uint64]struct{}, error) {
	val, err := s.cache.Get(ctx, blacklistKey)
	if err == nil && val != "" {
		var results map[uint64]struct{}
		err := json.Unmarshal([]byte(val), &results)
		if err == nil {
			return results, nil
		}
	}

	geoIds, err := s.contentClient.GetBlackListedHotels(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "can't get blacklisted hotels")
	}

	blacklist := make(map[uint64]struct{})
	for _, geoId := range geoIds {
		blacklist[geoId] = struct{}{}
	}

	marshal, err := json.Marshal(blacklist)
	if err == nil {
		s.cache.Set(ctx, blacklistKey, string(marshal), blacklistTTL)
	}
	return blacklist, nil
}

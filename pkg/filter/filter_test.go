package filter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/example/geo/models"
	coremodels "github.com/example/geo/suggest-core/models"
	"github.com/example/golang/log"
	"github.com/example/hotels-modules/cache"
)

type fakeContentClient struct {
	blacklist []uint64
}

func (fc *fakeContentClient) GetBlackListedHotels(ctx context.Context) ([]uint64, error) {
	return fc.blacklist, nil
}

func TestFilter_FilterSuggestion_WithFakes(t *testing.T) {
	fakeCache := cache.NewFake()

	fakeContentClient := &fakeContentClient{
		blacklist: []uint64{1, 3, 5},
	}

	f := NewFilter(log.For("test"), fakeContentClient, fakeCache)

	input := []coremodels.Suggestion{
		{GeoID: models.GeoID(1), GeoType: hotelGeoType},
		{GeoID: models.GeoID(2), GeoType: hotelGeoType},
		{GeoID: models.GeoID(3), GeoType: hotelGeoType},
		{GeoID: models.GeoID(4), GeoType: hotelGeoType},
		{GeoID: models.GeoID(5), GeoType: hotelGeoType},
		{GeoID: models.GeoID(6), GeoType: "locality"},
	}

	expected := []coremodels.Suggestion{
		{GeoID: models.GeoID(2), GeoType: hotelGeoType},
		{GeoID: models.GeoID(4), GeoType: hotelGeoType},
		{GeoID: models.GeoID(6), GeoType: "locality"},
	}

	ctx := context.Background()

	actual := f.FilterSuggestion(ctx, input)
	assert.Equal(t, expected, actual)
}

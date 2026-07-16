package geo

import (
	"context"
	"fmt"

	geocommon "github.com/example/geo/common"
	geoclient "github.com/example/geo/geo-client/v2/client"
	geomodels "github.com/example/geo/models"
	"github.com/pkg/errors"
)

type Client struct {
	source geoclient.DataSource
}

func NewClient(geoClient geoclient.DataSource) *Client {
	return &Client{source: geoClient}
}

func (c *Client) SetGeoClient(geoClient geoclient.DataSource) {
	c.source = geoClient
}

func (c *Client) GetLocalityByGeoIdAndGeoType(ctx context.Context, geoId uint64, geoType string) (geomodels.Locality, error) {
	switch geoType {
	case geomodels.GeoTypeLocality.String():
		return c.getLocalityByGeoId(ctx, geoId)
	case geomodels.GeoTypeHotel.String():
		return c.getLocalityByHotelId(ctx, geoId)
	case geomodels.GeoTypeAirport.String():
		return c.getLocalityByAirportId(ctx, geoId)
	default:
		return geomodels.Locality{}, fmt.Errorf("unsupported geo type: %s", geoType)
	}
}

func (c *Client) getLocalityByGeoId(ctx context.Context, geoId uint64) (geomodels.Locality, error) {
	locality, err := c.source.LocalityByID(ctx, geomodels.GeoID(geoId))
	if err != nil {
		return geomodels.Locality{}, errors.Wrap(err, "failed to get locality")
	}
	return locality, nil
}

func (c *Client) getLocalityByHotelId(ctx context.Context, geoId uint64) (geomodels.Locality, error) {
	hotel, err := c.source.HotelByID(ctx, geomodels.GeoID(geoId))
	if err != nil {
		return geomodels.Locality{}, errors.Wrap(err, "failed to get hotel")
	}

	locality, err := c.source.LocalityByHotel(ctx, hotel)
	if err != nil {
		return geomodels.Locality{}, errors.Wrap(err, "failed to get locality")
	}
	return locality, nil
}

func (c *Client) getLocalityByAirportId(ctx context.Context, geoId uint64) (geomodels.Locality, error) {
	airport, err := c.source.AirportByID(ctx, geomodels.GeoID(geoId))
	if err != nil {
		return geomodels.Locality{}, errors.Wrap(err, "failed to get airport")
	}

	locality, err := c.source.LocalityBelongedToAirport(ctx, airport, geomodels.RelationContextAvia)
	if err != nil {
		return geomodels.Locality{}, errors.Wrap(err, "failed to get belonged to airport locality")
	}

	return locality, nil
}

func (c *Client) GetAdmRegionNameByLocality(ctx context.Context, locality geomodels.Locality, locale string) (string, error) {
	admRegion, err := c.source.AdmRegionByLocality(ctx, locality)
	if err != nil {
		return "", errors.Wrap(err, "failed to get adm region name")
	}
	return admRegion.Name[locale], nil
}

func (c *Client) GetAdmDistrictNameByLocality(ctx context.Context, locality geomodels.Locality, locale string) (string, error) {
	admDistrict, err := c.source.AdmDistrictByLocality(ctx, locality)
	if err != nil {
		return "", errors.Wrap(err, "failed to get adm district name")
	}
	return admDistrict.Name[locale], nil
}

func (c *Client) GetCountryNameByLocality(ctx context.Context, locality geomodels.Locality, locale string) (string, error) {
	country, err := c.source.CountryByLocality(ctx, locality)
	if err != nil {
		return "", errors.Wrap(err, "failed to get country")
	}
	shortNames, err := c.source.AlternativeNamesByTypeAndGeoId(
		ctx,
		geomodels.AlternativeNameTypeShortName,
		geomodels.GeoID(country.ID.Uint64()),
		geocommon.LocaleEn,
	)
	if err == nil && len(shortNames) > 0 {
		return shortNames[0], nil
	}

	return country.Name[locale], nil
}

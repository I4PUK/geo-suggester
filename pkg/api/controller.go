package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"github.com/example/geo/common"
	geoproto "github.com/example/geo/contracts/geo-readonly-gateway/langs/go"
	geoclient "github.com/example/geo/geo-client/v2"
	memorycfg "github.com/example/geo/geo-client/v2/client/data_source/memory/config"
	geomodels "github.com/example/geo/models"
	"github.com/example/geo/suggest-core/elastic"
	elasticconfig "github.com/example/geo/suggest-core/elastic/config"
	"github.com/example/geo/suggest-core/migration"

	grpcserver "github.com/example/golang/grpc-server/v2"
	servercore "github.com/example/golang/grpc-server/v2/core"
	"github.com/example/golang/grpc-server/v2/middleware"
	httpserver "github.com/example/golang/http-server"
	"github.com/example/golang/readiness"
	contentpb "github.com/example/hotels-content/contracts/content/langs/go"
	"github.com/example/hotels-modules/cache"
	geo_suggest "github.com/example/hotels-search/contracts/geo-suggest/langs/go"
	"github.com/example/hotels-search/search/search/content"

	"github.com/example/geo-suggest/pkg/api/server"
	httpapi "github.com/example/geo-suggest/pkg/api/server/http"
	"github.com/example/geo-suggest/pkg/config"
	"github.com/example/geo-suggest/pkg/filter"
	"github.com/example/geo-suggest/pkg/geo"
	"github.com/example/geo-suggest/pkg/search"
	"github.com/example/geo-suggest/pkg/template"
)

const (
	DefaultServiceConfig = `{"loadBalancingConfig": [{"round_robin":{}}]}`
)

type Controller struct {
	storage    *elastic.Storage
	logger     *zerolog.Logger
	grpcServer *servercore.Server
	httpServer *httpserver.Server

	geoClient *geo.Client

	ready           *readiness.Controller
	cfg             *config.Config
	searchTemplates map[string]string
}

func NewController(logger *zerolog.Logger, cfg *config.Config) *Controller {
	ready := readiness.New()
	ready.SetNotReady()
	srv := httpserver.NewServer(ready, httpserver.Config{})
	return &Controller{
		logger:     logger,
		httpServer: srv,
		ready:      ready,
		cfg:        cfg,
	}
}

func (a *Controller) Start(ctx context.Context) error {
	if err := a.initStorage(ctx); err != nil {
		return err
	}
	a.grpcServer = grpcserver.NewServer().
		WithPretty().
		WithLoggingMiddlewares(middleware.LoggingOpts{
			LogRequestBody:  true,
			LogResponseBody: false,
		}).
		WithHandlingTimeHistogram()
	contentClient, err := a.getContentClient(ctx)
	if err != nil {
		return err
	}
	c, err := cache.NewRedis(a.logger, a.cfg.Redis.URL, a.cfg.Redis.Password)
	if err != nil {
		return errors.Wrap(err, "redis init issues")
	}

	if _, err := a.getGeoClient(ctx, a.logger); err != nil {
		return errors.Wrap(err, "geo-client init issues")
	}

	a.logger.Info().Msg("memory geo loaded")

	hotelFilter := filter.NewFilter(a.logger, contentClient, c)
	searchService := search.NewService(a.storage, a.logger, a.geoClient, hotelFilter, a.searchTemplates, a.cfg.TopHotelsInRegions)
	a.grpcServer.Register(func(srv *grpc.Server) {
		geo_suggest.RegisterSuggestServiceServer(srv, server.NewServer(searchService))
	})
	a.httpServer.Handle("/api/v1/suggest", httpapi.NewSuggestHandler(searchService)).Methods(http.MethodGet)
	a.httpServer.Handle("/api/v1/test", httpapi.NewTestHandler(searchService)).Methods(http.MethodGet)
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return a.httpServer.Start(ctx)
	})
	g.Go(func() error {
		return a.grpcServer.Start(ctx)
	})
	a.ready.SetReady()
	return g.Wait()
}

func (a *Controller) initStorage(ctx context.Context) error {
	cfg, err := elasticconfig.FromSpecs()
	if err != nil {
		return errors.Wrap(err, "init config")
	}
	a.storage = elastic.NewStorage(cfg)

	a.searchTemplates = a.loadSearchTemplate()
	for templateID, template := range a.searchTemplates {
		if err := migration.InstallSearchTemplate(ctx, a.storage, templateID, template); err != nil {
			return errors.Wrap(err, "installing new search template")
		}
	}
	return nil
}

func (a *Controller) loadSearchTemplate() map[string]string {
	out := make(map[string]string)
	filepath.Walk(a.cfg.ElasticSearchTemplateDIR, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.Contains(info.Name(), ".json") {
			if _, ok := out[template.GenerateTemplateID(info.Name())]; !ok {
				out[template.GenerateTemplateID(info.Name())] = path
			}
		}
		return nil
	})
	return out
}

func (a *Controller) getContentClient(ctx context.Context) (content.Client, error) {
	conn, err := grpc.DialContext(
		ctx,
		a.cfg.HotelContentEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(DefaultServiceConfig),
	)
	if err != nil {
		return nil, errors.Wrap(err, "can't connect to hotel content")
	}
	contentClient := contentpb.NewHotelContentClient(conn)
	return content.NewClient(contentClient, a.logger), nil
}

func (a *Controller) getGeoClient(ctx context.Context, logger *zerolog.Logger) (*geo.Client, error) {
	if a.geoClient == nil {
		memCfg := a.getGeoClientMemConfig()
		cfg := &geoproto.CommonConfig{}
		remoteConfig := geoclient.RemoteConfigFromMemoryConfig(memCfg, cfg)
		geoClientRemote, err := geoclient.NewRemote(ctx, remoteConfig)
		if err != nil {
			return nil, err
		}
		a.geoClient = geo.NewClient(geoClientRemote.DataSource)
		if a.cfg.GeoMemoryClient {
			go func() {
				geoMemClient, err := geoclient.NewMemory(ctx, memCfg)
				if err != nil {
					logger.Err(err).Msg("error on new geo memory client initialization")
					return
				}
				a.geoClient.SetGeoClient(geoMemClient.DataSource)
			}()
		}
	}
	return a.geoClient, nil
}

func (a *Controller) getGeoClientMemConfig() memorycfg.Config {
	return memorycfg.Config{
		Types: []geomodels.GeoType{
			geomodels.GeoTypeLocality,
			geomodels.GeoTypeCountry,
			geomodels.GeoTypeAdmRegion,
			geomodels.GeoTypeAdmDistrict,
			geomodels.GeoTypeAirport,
			geomodels.GeoTypeHotel,
		},
		Locality: memorycfg.LocalityConfig{
			LocalityConfig: &geoproto.LocalityConfig{
				Locales:      []string{common.LocaleEn},
				DuplicateIds: true,
			},
		},
		Country: memorycfg.CountryConfig{
			CountryConfig: &geoproto.CountryConfig{
				Locales:      []string{common.LocaleEn},
				DuplicateIds: true,
			},
			WithAltNamesIndex: true,
		},
		AdmRegion: memorycfg.AdmRegionConfig{
			AdmRegionConfig: &geoproto.AdmRegionConfig{
				Locales:      []string{common.LocaleEn},
				DuplicateIds: true,
			},
		},
		AdmDistrict: memorycfg.AdmDistrictConfig{
			AdmDistrictConfig: &geoproto.AdmDistrictConfig{
				Locales:      []string{common.LocaleEn},
				DuplicateIds: true,
			},
		},
		Airport: memorycfg.AirportConfig{
			AirportConfig: &geoproto.AirportConfig{
				Locales:      []string{common.LocaleEn},
				DuplicateIds: true,
			},
		},
		Hotel: memorycfg.HotelConfig{
			HotelConfig: &geoproto.HotelConfig{
				Locales:      []string{common.LocaleEn},
				DuplicateIds: true,
			},
		},
		UpdateTimeout: time.Hour,
		UpdateErrorAction: func(err error) {
			a.logger.Error().Err(err).Msg("geo client update error")
		},
	}
}

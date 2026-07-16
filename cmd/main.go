package main

import (
	"context"
	"math/rand"
	"os"
	"time"

	_ "go.uber.org/automaxprocs"
	"github.com/example/golang/context_os"
	"github.com/example/golang/envs"
	server "github.com/example/golang/http-server"
	"github.com/example/golang/log"
	"github.com/example/golang/resources/sentry"

	"github.com/example/geo-suggest/pkg/api"
	"github.com/example/geo-suggest/pkg/config"
)

const (
	ServiceName = "hotels-geo-suggest"
)

func main() {
	logger := log.For(ServiceName)
	server.CheckDisabledService()
	envs.Guard()
	rand.Seed(time.Now().UnixNano())

	cfg, err := config.ReadConfig()
	if err != nil {
		logger.Err(err).Msg("error initializing config")
		os.Exit(1)
	}
	c := api.NewController(logger, cfg)
	ctx := context_os.Context(context.Background())
	if err := c.Start(ctx); err != nil && err != context.Canceled {
		sentry.Client.CaptureErrorAndWait(err, nil)
		logger.Error().Err(err).Msg("error")
	}
	logger.Info().Msg("shutdown service")
}

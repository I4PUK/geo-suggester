package server

import (
	"context"

	"github.com/example/geo-suggest/pkg/models"
	"github.com/example/golang/locale"
	"github.com/example/golang/log"
	grpcmetadata "github.com/example/golang/requests/grpc_metadata"
	"github.com/example/golang/resources/sentry"
	geo_suggest "github.com/example/hotels-search/contracts/geo-suggest/langs/go"
	"github.com/example/locale"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/text/language"
	"google.golang.org/grpc/metadata"
)

type Server struct {
	geo_suggest.UnimplementedSuggestServiceServer

	service SearchService
	logger  *zerolog.Logger
}

func NewServer(service SearchService) *Server {
	return &Server{
		service: service,
		logger:  log.For("suggest"),
	}
}

func (s *Server) Suggest(ctx context.Context, request *geo_suggest.SuggestRequest) (*geo_suggest.SuggestResponse, error) {
	l := s.localizer(ctx, locale.EuropeanFallbacks)
	res, err := s.service.Suggest(ctx, models.Query{
		Query:    request.Query,
		Location: LocationFromProto(request.Location),
		Locale:   l.PrimaryLocale().String(),
		Limit:    uint(request.Limit),
	})
	if err != nil {
		return nil, errors.Wrap(err, "query error")
	}
	return &geo_suggest.SuggestResponse{Suggestions: SuggestionsToProto(res, l)}, nil
}

func (s *Server) localizer(ctx context.Context, fallbacks []language.Tag) *locale.L {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		sentry.ReportError(errors.New("can't read request metadata"))
	}
	l, err := locale.NewByMetadata(grpcmetadata.FromGRPCMetadata(md), fallbacks)
	if err != nil {
		sentry.ReportError(err)
	}
	return l
}

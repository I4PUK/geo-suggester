package server

import (
	"context"

	coremodels "github.com/example/geo/suggest-core/models"
	"github.com/example/geo-suggest/pkg/models"
)

type SearchService interface {
	Suggest(ctx context.Context, q models.Query) ([]coremodels.Suggestion, error)
}

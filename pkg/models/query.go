package models

import (
	coremodels "github.com/example/geo/suggest-core/models"
)

type Query struct {
	Query    string
	Location *coremodels.Location
	Locale   string
	Limit    uint
}

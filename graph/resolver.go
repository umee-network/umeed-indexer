package graph

import (
	"github.com/rs/zerolog"
	"github.com/umee-network/umeed-indexer/database"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.
const DefaultChainID = "umee-1"

type Resolver struct {
	db     database.Database
	logger zerolog.Logger
}

// NewResolver returns a new resolver.
func NewResolver(db database.Database, logger zerolog.Logger) *Resolver {
	return &Resolver{
		db:     db,
		logger: logger,
	}
}

func defaultChainID(chainID *string) string {
	if chainID != nil {
		return *chainID
	}
	return DefaultChainID
}

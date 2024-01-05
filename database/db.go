package database

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/umee-network/umeed-indexer/database/firebase"
)

// TypeDB defines the databases available for indexing.
type TypeDB uint8

const (
	// Firebase is the default DB for this indexer.
	Firebase TypeDB = iota + 1
	// TODO: add in memory for testing.
)

// Database defines the exported functions of the database.
type Database interface {
	/*
		Basic
	*/

	// Close closes the needed connections.
	Close() error
	// DeleteAll inside the database.
	DeleteAll(ctx context.Context) error

	/*
		Chain Data
	*/

	// DeleteChainData delete the chain data and all of its structures inside.
	DeleteChainData(ctx context.Context, chainID string) error
	// UpsertChainHeader updates the chain id and current height of chain.
	UpsertChainHeader(ctx context.Context, chainID string, height int) error
}

// NewDB returns a new database instance based on the specified type.
// It returns an error if the database type is unsupported.
func NewDB(typeDB TypeDB, ctx context.Context, logger zerolog.Logger) (Database, error) {
	switch typeDB {
	case Firebase:
		return loadFirebase(ctx, logger)
	default:
		return nil, fmt.Errorf("unsupported database type: %v", typeDB)
	}
}

// loadFirebase checks if there is env set for emulator, if it is loads firebase without credentials
// otherwise it looks for credentials to run
func loadFirebase(ctx context.Context, logger zerolog.Logger) (Database, error) {
	firebaseEmulator := os.Getenv(firebase.EnvFirebaseEmulator)
	if len(firebaseEmulator) > 0 {
		fmt.Printf(
			"\nEnv %s detected as %s, API running on firestore emulator",
			firebase.EnvFirebaseEmulator,
			firebaseEmulator,
		)
		return firebase.New(ctx, logger)
	}

	opt, err := firebase.LoadCredential()
	if err != nil {
		return nil, errors.Join(err, errors.New("Failed to load Firebase credentials"))
	}

	db, err := firebase.New(ctx, logger, opt)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to initialize Firebase database"))
	}

	return db, err
}

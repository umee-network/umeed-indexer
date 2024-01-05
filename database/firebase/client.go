package firebase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/rs/zerolog"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	// tests should set this to true if the delete messages get spammy
	MUTE_DELETE = true
)

const (
	EnvFirebaseCredentialsJSON = "FIREBASE_CREDENTIALS_JSON"
	EnvFirebaseEmulator        = "FIRESTORE_EMULATOR_HOST"
	EnvFirebaseProjectID       = "FIREBASE_PROJECT_ID"
)

// Database stores the information needed to call getters and setters on firebase.
type Database struct {
	// App is the primary Firebase app instance.
	app *firebase.App
	// Fs is the firestore instance to access the database.
	Fs *firestore.Client
	// Auth is the Firebase Auth client.
	auth *auth.Client
	// logger
	logger zerolog.Logger
}

// New returns a new firebase struture.
func New(ctx context.Context, logger zerolog.Logger, opts ...option.ClientOption) (db *Database, err error) {
	cfg := loadConfig()

	// Use a service account
	app, err := firebase.NewApp(ctx, cfg, opts...)
	if err != nil {
		return nil, errors.Join(err, errors.New("Failed to initialize Firebase app"))
	}

	appAuth, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("error initializing auth client: %v\n", err)
		return nil, errors.Join(err, errors.New("Failed to initialize Firebase auth client"))
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("Failed to initialize Firestore client"))
	}

	return &Database{
		app:    app,
		Fs:     client,
		auth:   appAuth,
		logger: logger.With().Str("database", "firebase").Logger(),
	}, nil
}

// Close closes any open connection it might have.
func (db *Database) Close() error {
	if err := db.Fs.Close(); err != nil {
		return errors.Join(err, errors.New("Failed to close Firestore client"))
	}
	return nil
}

// RunTransaction run a transaction, all reads must come first than any write in db.
func (db *Database) RunTransaction(
	ctx context.Context,
	f func(context.Context, *firestore.Transaction) error,
	opts ...firestore.TransactionOption,
) (err error) {
	return db.Fs.RunTransaction(ctx, f, opts...)
}

// DeleteAll inside the database.
func (db *Database) DeleteAll(ctx context.Context) error {
	collIter := db.Fs.Collections(ctx)
	for {
		coll, err := collIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		if err := db.DeleteCollection(ctx, coll); err != nil {
			return err
		}
	}
	return nil
}

// DeleteChainData delete the chain data and all of its structures inside.
func (d Database) DeleteChainData(ctx context.Context, chainID string) error {
	chainDoc := d.Fs.Collection(CollChain).Doc(chainID)

	// calls this func again for the collections inside of it.
	chainCollIter := chainDoc.Collections(ctx)
	for {
		collInsideChainDoc, err := chainCollIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		// deletes denom metadata at /chains/${chainID}/denoms-metadata
		// deletes pools at /chains/${chainID}/pools
		if err := d.DeleteCollection(ctx, collInsideChainDoc); err != nil {
			return err
		}
	}

	_, err := chainDoc.Delete(ctx)
	return err
}

// DeleteCollection delete the docs inside an collection.
func (db *Database) DeleteCollection(ctx context.Context, coll *firestore.CollectionRef) error {
	bulkWriter := db.Fs.BulkWriter(ctx)

	for {
		// Get a batch of documents
		iter := coll.Documents(ctx)
		numDeleted := 0

		// Iterate through the documents, adding
		// a delete operation for each one to the BulkWriter.
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			if !MUTE_DELETE {
				fmt.Printf("reading %s\n", doc.Ref.Path)
			}

			// calls this func again for the collections inside of it.
			docCollIter := doc.Ref.Collections(ctx)
			for {
				collInsideDoc, err := docCollIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return err
				}
				if err := db.DeleteCollection(ctx, collInsideDoc); err != nil {
					return err
				}
				if !MUTE_DELETE {
					fmt.Printf("deleted collection \"%s\"  inside doc - \"%s\"\n", collInsideDoc.Path, doc.Ref.Path)
				}
			}

			_, err = bulkWriter.Delete(doc.Ref)
			if err != nil {
				fmt.Printf("error deleting %s = %s", doc.Ref.ID, err)
				continue
			}
			numDeleted++
		}

		// If there are no documents to delete,
		// the process is over.
		if numDeleted == 0 {
			bulkWriter.End()
			break
		}

		bulkWriter.Flush()
	}
	if !MUTE_DELETE {
		fmt.Printf("deleted collection \"%s\"\n", coll.Path)
	}

	return nil
}

// LoadCredential checks if there is the env of an config full path or
// if it doesn't exists, tries to load from an json base64 encoded.
func LoadCredential() (option.ClientOption, error) {
	var (
		credentials []byte
		err         error
	)

	// Try to get credentials from environment variable first
	credentialsFromEnv := os.Getenv(EnvFirebaseCredentialsJSON)
	if credentialsFromEnv != "" {
		credentials = []byte(credentialsFromEnv)
	} else {
		// If not found in environment, read from firebase.json
		credentials, err = os.ReadFile("firebase.json")
		if err != nil {
			fmt.Printf("Failed to read firebase.json and no %s found: %v", EnvFirebaseCredentialsJSON, err)
			return nil, errors.Join(err, errors.New("Failed to read firebase.json"))
		}
	}

	return option.WithCredentialsJSON(credentials), nil
}

// loadConfig returns the firebase config
func loadConfig() *firebase.Config {
	return &firebase.Config{
		ProjectID: getEnvOrDefault(EnvFirebaseProjectID, "umeed-indexer"),
	}
}

// getEnvOrDefault loads the env variable, if it is empty, returns the default.
func getEnvOrDefault(envVarName, defaultForEnv string) string {
	evnValue := os.Getenv(envVarName)
	if len(evnValue) == 0 {
		return defaultForEnv
	}
	return evnValue
}

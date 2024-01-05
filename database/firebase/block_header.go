package firebase

import (
	"context"

	"cloud.google.com/go/firestore"
)

const (
	CollChain = "chains"
)

// UpsertChainHeader updates the chain id and current height of chain.
func (d Database) UpsertChainHeader(ctx context.Context, chainID string, height int) error {
	_, err := d.Fs.Collection(CollChain).Doc(chainID).Set(ctx, map[string]any{
		"chainID": chainID,
		"height":  height,
	}, firestore.MergeAll)
	return err
}

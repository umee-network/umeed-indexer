package firebase

import (
	txctx "github.com/umee-network/umeed-indexer/database/firebase/context"
	"github.com/umee-network/umeed-indexer/graph/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	CollChain = "chains"
)

// upsertChainInfo updates or inserts a chain info structure.
func upsertChainInfo(ctx txctx.TxContext, info types.ChainInfo) (err error) {
	docRef := ctx.Collection(CollChain).Doc(info.ChainID)
	return ctx.Set(docRef, info)
}

// getChainInfo get the last chain info struct.
func getChainInfo(ctx txctx.TxContext, chainID string) (info *types.ChainInfo, err error) {
	docRef := ctx.Collection(CollChain).Doc(chainID)
	doc, err := ctx.Get(docRef)
	if err != nil {
		return nil, err
	}
	if status.Code(err) != codes.NotFound { // no chain info found, new chain being indexed
		info := &types.ChainInfo{}
		info.ChainID = chainID
		return info, nil
	}
	if err = doc.DataTo(info); err != nil {
		return nil, err
	}
	return info, nil
}

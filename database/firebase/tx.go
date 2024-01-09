package firebase

import (
	txctx "github.com/umee-network/umeed-indexer/database/firebase/context"
	"github.com/umee-network/umeed-indexer/graph/types"
)

const (
	CollTransactions = "transactions"
)

// addTx adds a new tx structure.
func addTx(ctx txctx.TxContext, chainID string, tx types.IndexedTx) (err error) {
	collTxs := ctx.Collection(CollChain).Doc(chainID).Collection(CollTransactions)
	docRef := collTxs.NewDoc()
	return ctx.Set(docRef, tx)
}

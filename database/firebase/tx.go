package firebase

import (
	"cloud.google.com/go/firestore"
	txctx "github.com/umee-network/umeed-indexer/database/firebase/context"
	"github.com/umee-network/umeed-indexer/graph/types"
	"google.golang.org/api/iterator"
)

const (
	CollTransactions = "transactions"
)

// addTx adds a new tx structure.
func addTx(ctx txctx.TxContext, chainID string, tx types.IndexedTx) (err error) {
	collTxs := collTxs(ctx, chainID)
	docRef := collTxs.NewDoc()
	return ctx.Set(docRef, tx)
}

func getTxLiquidade(ctx txctx.TxContext, chainID, borrower string) (txs []*types.IndexedTx, err error) {
	collTxs := collTxs(ctx, chainID)

	query := collTxs.Query.WhereEntity(
		firestore.OrFilter{
			Filters: []firestore.EntityFilter{
				firestore.PropertyPathFilter{Path: []string{"msgLiquidate", "borrower"}, Operator: "==", Value: borrower},
				firestore.PropertyPathFilter{Path: []string{"msgLeverageLiquidate", "borrower"}, Operator: "==", Value: borrower},
			},
		},
	)

	iter := query.Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return txs, err
		}

		var tx types.IndexedTx
		if err := doc.DataTo(&tx); err != nil {
			return nil, err
		}
		txs = append(txs, &tx)
	}
	return txs, nil
}

func collTxs(ctx txctx.TxContext, chainID string) (collTxs *firestore.CollectionRef) {
	return ctx.Collection(CollChain).Doc(chainID).Collection(CollTransactions)
}

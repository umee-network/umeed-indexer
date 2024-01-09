package firebase

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	txctx "github.com/umee-network/umeed-indexer/database/firebase/context"
	"github.com/umee-network/umeed-indexer/graph/types"
)

// UpsertChainInfo updates or inserts a chain info structure.
func (db *Database) UpsertChainInfo(ctx context.Context, info types.ChainInfo) (err error) {
	err = db.RunTransaction(
		ctx, func(ctx context.Context, t *firestore.Transaction) error {
			tctx := txctx.New(ctx, time.Now(), t, db.Fs)
			return upsertChainInfo(tctx, info)
		},
	)
	return err
}

// GetChainInfo returns the last chainInfo.
func (db *Database) GetChainInfo(ctx context.Context, chainID string) (info *types.ChainInfo, err error) {
	err = db.RunTransaction(
		ctx, func(ctx context.Context, t *firestore.Transaction) error {
			tctx := txctx.New(ctx, time.Now(), t, db.Fs)
			info, err = getChainInfo(tctx, chainID)
			return err
		},
	)
	return info, err
}

func (db *Database) StoreMsgLiquidate(ctx context.Context, chainID string, txHash []byte, blockHeight int, msg types.MsgLiquidate) (err error) {
	err = db.RunTransaction(
		ctx, func(ctx context.Context, t *firestore.Transaction) error {
			tctx := txctx.New(ctx, time.Now(), t, db.Fs)
			return addTx(tctx, chainID, types.IndexedTx{
				TxHash:       string(txHash),
				ProtoMsgName: types.MsgNameLiquidate,
				BlockHeight:  blockHeight,
				MsgLiquidate: &msg,
			})
		},
	)
	return err
}

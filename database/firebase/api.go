package firebase

import (
	"context"

	"cloud.google.com/go/firestore"
	txctx "github.com/umee-network/umeed-indexer/database/firebase/context"
	"github.com/umee-network/umeed-indexer/graph/types"
)

// UpsertChainInfo updates or inserts a chain info structure.
func (db *Database) UpsertChainInfo(ctx context.Context, info types.ChainInfo) (err error) {
	err = db.RunTransaction(
		ctx, func(ctx context.Context, t *firestore.Transaction) error {
			tctx := txctx.Now(ctx, t, db.Fs)
			return upsertChainInfo(tctx, info)
		},
	)
	return err
}

// GetChainInfo returns the last chainInfo.
func (db *Database) GetChainInfo(ctx context.Context, chainID string) (info *types.ChainInfo, err error) {
	err = db.RunTransaction(
		ctx, func(ctx context.Context, t *firestore.Transaction) error {
			tctx := txctx.Now(ctx, t, db.Fs)
			info, err = getChainInfo(tctx, chainID)
			return err
		},
	)
	return info, err
}

// StoreMsgLiquidate stores a new msgliquidate updating the CosmosMsgIndexed.
func (db *Database) StoreMsgLiquidate(ctx context.Context, chainInfo types.ChainInfo, blockHeight, blockTimeUnix int, txHash string, msg types.MsgLiquidate) (err error) {
	err = db.RunTransaction(
		ctx, func(ctx context.Context, t *firestore.Transaction) error {
			tctx := txctx.Now(ctx, t, db.Fs)
			err = addTx(tctx, chainInfo.ChainID, types.IndexedTx{
				TxHash:        txHash,
				ProtoMsgName:  types.MsgNameLiquidate,
				BlockHeight:   blockHeight,
				BlockTimeUnix: blockTimeUnix,
				MsgLiquidate:  &msg,
			})
			if err != nil {
				return err
			}

			return upsertChainInfo(tctx, chainInfo)
		},
	)
	return err
}

// StoreMsgLeverageLiquidate stores a new MsgLeverageLiquidate updating the CosmosMsgIndexed.
func (db *Database) StoreMsgLeverageLiquidate(ctx context.Context, chainInfo types.ChainInfo, blockHeight, blockTimeUnix int, txHash string, msg types.MsgLeverageLiquidate) (err error) {
	err = db.RunTransaction(
		ctx, func(ctx context.Context, t *firestore.Transaction) error {
			tctx := txctx.Now(ctx, t, db.Fs)
			err = addTx(tctx, chainInfo.ChainID, types.IndexedTx{
				TxHash:               txHash,
				ProtoMsgName:         types.MsgNameLiquidate,
				BlockHeight:          blockHeight,
				BlockTimeUnix:        blockTimeUnix,
				MsgLeverageLiquidate: &msg,
			})
			if err != nil {
				return err
			}

			return upsertChainInfo(tctx, chainInfo)
		},
	)
	return err
}

// GetLiquidateMsgs returns all the msgs liquidate filtering by the borrower.
func (db *Database) GetLiquidateMsgs(ctx context.Context, chainID string, borrower string) (txs []*types.IndexedTx, err error) {
	txs = make([]*types.IndexedTx, 0)
	err = db.RunTransaction(
		ctx, func(ctx context.Context, t *firestore.Transaction) error {
			tctx := txctx.Now(ctx, t, db.Fs)
			txs, err = getTxLiquidade(tctx, chainID, borrower)
			return nil
		},
	)
	return txs, err
}

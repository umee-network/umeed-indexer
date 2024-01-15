package idx

import (
	"context"
	"encoding/hex"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/gogoproto/proto"
	lvgtypes "github.com/umee-network/umee/v6/x/leverage/types"
	"github.com/umee-network/umeed-indexer/graph/types"
)

// HandleNewBlock handles the receive of new block from the chain.
func (i *Indexer) HandleNewBlock(ctx context.Context, blk *tmtypes.Block) error {
	i.b.SetChainHeader(blk)
	i.logger.Info().Int64("height", blk.Height).Msg("new block received")

	// since it is a new block, updates the chain info base information
	i.chainInfo.UpdateFromBlock(blk)

	// and continues to handle a block normally.
	return i.HandleBlock(ctx, blk)
}

// HandleBlock handles the receive of an block from the chain.
func (i *Indexer) HandleBlock(ctx context.Context, blk *tmtypes.Block) error {
	for _, tx := range blk.Data.Txs {
		if err := i.HandleTx(ctx, int(blk.Header.Height), int(blk.Time.Unix()), tx); err != nil {
			i.logger.Err(err).Int64("height", blk.Height).Msg("error handling block")
			continue
		}
	}

	return i.chainInfo.Execute(func(info *types.ChainInfo) error {
		info.IndexBlockHeight(int(blk.Height))
		return i.db.UpsertChainInfo(ctx, *info)
	})
}

// HandleTx handles the receive of new Tx from the chain.
func (i *Indexer) HandleTx(ctx context.Context, blockHeight, blockTimeUnix int, tmTx tmtypes.Tx) error {
	tx, err := i.b.DecodeTx(tmTx)
	if err != nil {
		i.logger.Err(err).Msg("error decoding Tx")
		return err
	}

	txHash := tmTx.Hash()
	txMsgs := tx.GetMsgs()

	for _, msg := range txMsgs {
		if err := i.HandleMsg(ctx, blockHeight, blockTimeUnix, tmTx, txHash, msg); err != nil {
			i.logger.Err(err).Msg("error handling msg")
			continue
		}
	}
	return nil
}

// HandleMsg handles the receive of new msg from the chain Tx.
func (i *Indexer) HandleMsg(ctx context.Context, blkHeight, blockTimeUnix int, tmTx tmtypes.Tx, txHash []byte, msg proto.Message) error {
	msgName := proto.MessageName(msg)

	switch msgName {
	case types.MsgNameLiquidate:
		msgLiq, ok := msg.(*lvgtypes.MsgLiquidate)
		if !ok {
			i.logger.Error().Str("messageName", msgName).Msg("not able to parse into *lvgtypes.MsgLiquidate")
			return nil
		}

		i.logger.Debug().Msg("storing msg liquidate")
		return i.indexMsg(ctx, msgName, blkHeight, tmTx, func(info *types.ChainInfo) error {
			return i.db.StoreMsgLiquidate(ctx, *info, blkHeight, blockTimeUnix, hex.EncodeToString(tmTx.Hash()), types.ParseTxLiquidate(msgLiq))
		})
	case types.MsgNameLeveragedLiquidate:
		msgLevLiq, ok := msg.(*lvgtypes.MsgLeveragedLiquidate)
		if !ok {
			i.logger.Error().Str("messageName", msgName).Msg("not able to parse into *lvgtypes.MsgLeveragedLiquidate")
			return nil
		}

		i.logger.Debug().Msg("storing msg leverage liquidate")
		return i.indexMsg(ctx, msgName, blkHeight, tmTx, func(info *types.ChainInfo) error {
			return i.db.StoreMsgLeverageLiquidate(ctx, *info, blkHeight, blockTimeUnix, hex.EncodeToString(tmTx.Hash()), types.ParseTxLeverageLiquidate(msgLevLiq))
		})
	default:
		// i.logger.Debug().Str("messageName", msgName).Msg("no handle for msg")
	}
	return nil
}

// indexMsg verifies if there is a need to stores that msg in the tx and if there is, verify if the tx was already processed.
func (i *Indexer) indexMsg(ctx context.Context, msgName string, blkHeight int, tmTx tmtypes.Tx, store func(info *types.ChainInfo) error) error {
	return i.chainInfo.Execute(func(info *types.ChainInfo) error {
		if !types.NeedsToIndexForMsg(msgName, info.CosmosMsgs, blkHeight) {
			i.logger.Debug().Str("messageName", msgName).Int("height", blkHeight).Msg("no need to store msg for this block height")
			return nil
		}

		if err := i.b.CheckTx(ctx, tmTx); err != nil {
			i.logger.Err(err).Str("messageName", msgName).Str("txHash", hex.EncodeToString(tmTx.Hash())).Int("height", blkHeight).Msg("tx failed, no need to store")
			return err
		}

		i.logger.Debug().Str("messageName", msgName).Str("txHash", hex.EncodeToString(tmTx.Hash())).Int("height", blkHeight).Msg("storing msg into db")
		return store(info)
	})
}

package idx

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/rs/zerolog"
	lvgtypes "github.com/umee-network/umee/v6/x/leverage/types"
	"github.com/umee-network/umeed-indexer/database"
	"github.com/umee-network/umeed-indexer/graph/types"
)

const (
	IDX_BLOCKS_PER_MINUTE = 100
)

// Indexer struct responsible for calling blockchain rpc/websocket for data and
// storing that into the database.
type Indexer struct {
	b         Blockchain
	db        database.Database
	logger    zerolog.Logger
	chainInfo types.ChainInfo
	// defines the lest block that the node has available in his store,
	// usually nodes do not keep all the blocks forever.
	lowestBlockHeightAvailableOnNode int
}

// NewIndexer returns a new indexer struct with open connections.
func NewIndexer(ctx context.Context, b Blockchain, db database.Database, logger zerolog.Logger) (*Indexer, error) {
	i := &Indexer{
		b:                                b,
		db:                               db,
		logger:                           logger.With().Str("package", "idx").Logger(),
		lowestBlockHeightAvailableOnNode: 1,
	}
	return i, i.onStart(ctx)
}

// Index starts to index transactions.
func (i *Indexer) Index(ctx context.Context) error {
	newBlock, err := i.b.SubscribeNewBlock(ctx)
	if err != nil {
		return err
	}

	return i.IndexCases(ctx, newBlock)
}

// IndexCases handle all the cases for the indexer.
func (i *Indexer) IndexCases(
	ctx context.Context,
	cNewBlock <-chan *tmtypes.Block,
) error {
	oneMin := time.NewTicker(time.Second)
	defer oneMin.Stop()

	for {
		select {
		// only closes the connections if the context is done.
		case <-ctx.Done():
			return i.Close(ctx)

		case blk := <-cNewBlock: // listen to new blocks being produced.
			if err := i.HandleNewBlock(ctx, blk); err != nil {
				i.logger.Err(err).Msg("error handling block")
			}

		case <-oneMin.C: // every minute. Tries to index from old blocks, if needed.
			i.logger.Info().Msg("One minute passed")
			i.IndexOldBlocks(ctx)
		}
	}
}

// IndexOldBlocks checks if it is needed to index old blocks and index them as needed.
func (i *Indexer) IndexOldBlocks(ctx context.Context) {
	if len(i.chainInfo.CosmosMsgs) == 0 { // safe check that we need to have some cosmos msg.
		return
	}

	lowestBlock := i.chainInfo.LowestBlockHeightToIndex(i.lowestBlockHeightAvailableOnNode)
	heighestBlock := lowestBlock + IDX_BLOCKS_PER_MINUTE
	// if the lowest block needed to index is not {IDX_BLOCKS_PER_MINUTE} behind the current
	// block, no need to try to index, wait until it is old enough.
	if heighestBlock > i.chainInfo.LastBlockHeightReceived {
		i.logger.Info().Int("fromBlock", lowestBlock).Int("ToBlock", heighestBlock).Msg("no need to index old blocks")
		return
	}

	blockHeight := lowestBlock
	blk, minimumNodeBlkHeight, err := i.b.Block(ctx, int64(blockHeight))
	if err != nil {
		i.logger.Err(err).Int("blockHeight", blockHeight).Msg("error getting old block from blockchain")
		return
	}

	if blk == nil && minimumNodeBlkHeight != 0 {
		i.logger.Info().Int("blockHeight", blockHeight).Int("minimumNodeBlkHeight", minimumNodeBlkHeight).Msg("initial block height not available on node")
		// in this case we should continue to index from the given height.
		i.lowestBlockHeightAvailableOnNode = minimumNodeBlkHeight
		i.IndexOldBlocks(ctx)
		return
	}

	if err := i.HandleBlock(ctx, blk); err != nil {
		i.logger.Err(err).Int("blockHeight", blockHeight).Msg("error handling old block")
	}
	i.IndexBlocksFromTo(ctx, lowestBlock+1, heighestBlock)
}

func (i *Indexer) IndexBlocksFromTo(ctx context.Context, from, to int) {
	for blockHeight := from; blockHeight < to; blockHeight++ {
		if !i.chainInfo.NeedsToIndex(blockHeight) {
			continue
		}
		i.logger.Debug().Int("blockHeight", blockHeight).Msg("indexing old block")

		blk, _, err := i.b.Block(ctx, int64(blockHeight))
		if err != nil {
			i.logger.Err(err).Int("blockHeight", blockHeight).Msg("error getting old block from blockchain")
			continue
		}

		if err := i.HandleBlock(ctx, blk); err != nil {
			i.logger.Err(err).Int("blockHeight", blockHeight).Msg("error handling old block")
		}
	}
}

// HandleNewBlock handles the receive of new block from the chain.
func (i *Indexer) HandleNewBlock(ctx context.Context, blk *tmtypes.Block) error {
	i.b.SetChainHeader(blk)
	i.logger.Info().Int64("height", blk.Height).Msg("new block received")
	return i.HandleBlock(ctx, blk)
}

// HandleBlock handles the receive of an block from the chain.
func (i *Indexer) HandleBlock(ctx context.Context, blk *tmtypes.Block) error {
	for _, tx := range blk.Data.Txs {
		if err := i.HandleTx(ctx, int(blk.Header.Height), tx); err != nil {
			return err
		}
	}

	i.chainInfo.BlockHeightIndexed(int(blk.Height))
	return i.UpsertChainInfo(ctx)
}

// UpsertChainInfo updates the chain info.
func (i *Indexer) UpsertChainInfo(ctx context.Context) error {
	return i.db.UpsertChainInfo(ctx, i.chainInfo)
}

// HandleTx handles the receive of new Tx from the chain.
func (i *Indexer) HandleTx(ctx context.Context, blockHeight int, tmTx tmtypes.Tx) error {
	tx, err := i.b.DecodeTx(tmTx)
	if err != nil {
		i.logger.Err(err).Msg("error decoding Tx")
		return err
	}

	txHash := tmTx.Hash()
	txMsgs := tx.GetMsgs()

	for _, msg := range txMsgs {
		if err := i.HandleMsg(ctx, blockHeight, txHash, msg); err != nil {
			i.logger.Err(err).Msg("error handling msg")
			continue
		}
	}
	return nil
}

// HandleMsg handles the receive of new msg from the chain Tx.
func (i *Indexer) HandleMsg(ctx context.Context, blockHeight int, txHash []byte, msg proto.Message) error {
	msgName := proto.MessageName(msg)
	switch msgName {
	case types.MsgNameLiquidate:
		msgLiq, ok := msg.(*lvgtypes.MsgLiquidate)
		if ok {
			fmt.Printf("\n is msg liquidate %+v", msg.String())
			fmt.Printf("\n msgLiq%+v", msgLiq)
			return i.db.StoreMsgLiquidate(ctx, i.chainInfo.ChainID, txHash, blockHeight, types.ParseTxLeverageLiquidate(msgLiq))
		}
	default:
		i.logger.Debug().Str("messageName", msgName).Msg("no handle for msg")
	}
	return nil
}

// onStart loads the starter data into blockchain.
func (i *Indexer) onStart(ctx context.Context) error {
	if err := i.loadChainHeader(ctx); err != nil {
		return err
	}

	// from which old block should index.
	return nil
}

// loadChainHeader queries the chain by the last block height and sets the chain ID inside
// the blockchain structure.
func (i *Indexer) loadChainHeader(ctx context.Context) error {
	chainID, height, err := i.b.ChainHeader()
	if err != nil {
		fmt.Printf("\nerr on loadChainHeader %s", err.Error())
		return err
	}
	info, err := i.db.GetChainInfo(ctx, chainID)
	if err != nil {
		i.logger.Err(err).Msg("error loading chain info")
		return err
	}
	info.LastBlockHeightReceived = int(height)
	i.chainInfo = *info
	return i.db.UpsertChainInfo(ctx, *info)
}

// Close closes all the open connections.
func (i *Indexer) Close(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return i.b.Close(ctx)
	})
	g.Go(func() error {
		return i.db.Close()
	})

	return g.Wait()
}

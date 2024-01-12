package idx

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/rs/zerolog"
	"github.com/umee-network/umeed-indexer/database"
	"github.com/umee-network/umeed-indexer/graph/types"
)

const (
	IDX_BLOCKS_PER_MINUTE = 100
)

// Indexer struct responsible for calling blockchain rpc/websocket for data and
// storing that into the database.
type Indexer struct {
	b      Blockchain
	db     database.Database
	logger zerolog.Logger

	chainInfo SafeChainInfo

	// defines the lest block that the node has available in his store,
	// usually nodes do not keep all the blocks forever.
	lowestBlockHeightAvailableOnNode int
}

// NewIndexer returns a new indexer struct with open connections.
func NewIndexer(ctx context.Context, b Blockchain, db database.Database, logger zerolog.Logger, startFromBlockHeight int) (*Indexer, error) {
	i := &Indexer{
		b:                                b,
		db:                               db,
		logger:                           logger.With().Str("package", "idx").Logger(),
		lowestBlockHeightAvailableOnNode: startFromBlockHeight,
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
	oneMin := time.NewTicker(time.Second * 60)
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
			i.logger.Info().Msgf("One minute passed")
			go i.IndexOldBlocks(ctx)
		}
	}
}

// IndexOldBlocks checks if it is needed to index old blocks and index them as needed.
func (i *Indexer) IndexOldBlocks(ctx context.Context) {
	cosmosMsgs, lastBlockHeightReceived := i.chainInfo.Copy()
	if len(cosmosMsgs) == 0 { // safe check that we need to have some cosmos msg.
		return
	}

	lowestBlock := types.LowestBlockHeightToIndex(cosmosMsgs, i.lowestBlockHeightAvailableOnNode)
	heighestBlock := lowestBlock + IDX_BLOCKS_PER_MINUTE
	// if the lowest block needed to index is not {IDX_BLOCKS_PER_MINUTE} behind the current
	// block, no need to try to index, wait until it is old enough.
	if heighestBlock > lastBlockHeightReceived {
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
	i.IndexBlocksFromTo(ctx, lowestBlock+1, heighestBlock, cosmosMsgs)
}

// IndexBlocksFromTo index blocks from specific heights.
func (i *Indexer) IndexBlocksFromTo(ctx context.Context, from, to int, cosmosMsgs []*types.CosmosMsgIndexed) {
	var wg sync.WaitGroup
	mapBlockByHeight := make(map[int]*tmtypes.Block)

	for blockHeight := from; blockHeight < to; blockHeight++ {
		blockHeight := blockHeight
		if !types.NeedsToIndex(cosmosMsgs, blockHeight) {
			continue
		}
		i.logger.Debug().Int("blockHeight", blockHeight).Msg("indexing old block")

		wg.Add(1) // what takes a lot of time is querying blocks from node
		go func(blockHeight int) {
			defer wg.Done()
			blk, _, err := i.b.Block(ctx, int64(blockHeight))
			if err != nil {
				i.logger.Err(err).Int("blockHeight", blockHeight).Msg("error getting old block from blockchain")
				return
			}
			mapBlockByHeight[blockHeight] = blk

		}(blockHeight)
	}

	wg.Wait()
	for blockHeight := from; blockHeight < to; blockHeight++ {
		blk, ok := mapBlockByHeight[blockHeight]
		if !ok {
			continue
		}

		if err := i.HandleBlock(ctx, blk); err != nil {
			i.logger.Err(err).Int("blockHeight", blockHeight).Msg("error handling old block")
		}
	}

}

// UpsertChainInfo updates the chain info.
func (i *Indexer) UpsertChainInfo(ctx context.Context) error {
	return i.chainInfo.Execute(func(info *types.ChainInfo) error {
		return i.db.UpsertChainInfo(ctx, *info)
	})
}

// onStart loads the starter data into blockchain.
func (i *Indexer) onStart(ctx context.Context) error {
	return i.loadChainHeader(ctx)
}

// loadChainHeader queries the chain by the last block height and sets the chain ID inside
// the blockchain structure.
func (i *Indexer) loadChainHeader(ctx context.Context) error {
	chainID, height, err := i.b.ChainHeader()
	if err != nil {
		i.logger.Err(err).Msg("error loading chain header")
		return err
	}
	info, err := i.db.GetChainInfo(ctx, chainID)
	if err != nil {
		i.logger.Err(err).Msg("error loading chain info")
		return err
	}
	info.LastBlockHeightReceived = int(height)
	i.chainInfo = *NewSafeChainInfo(info)
	return i.UpsertChainInfo(ctx)
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

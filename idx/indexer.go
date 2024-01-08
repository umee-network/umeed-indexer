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
	oracletypes "github.com/umee-network/umee/v6/x/oracle/types"
	"github.com/umee-network/umeed-indexer/database"
)

// Indexer struct responsible for calling blockchain rpc/websocket for data and
// storing that into the database.
type Indexer struct {
	b      Blockchain
	db     database.Database
	logger zerolog.Logger
}

// NewIndexer returns a new indexer struct with open connections.
func NewIndexer(ctx context.Context, b Blockchain, db database.Database, logger zerolog.Logger) (*Indexer, error) {
	i := &Indexer{
		b:      b,
		db:     db,
		logger: logger.With().Str("package", "idx").Logger(),
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
	oneMin := time.NewTicker(time.Minute)
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

		case <-oneMin.C: // every minute.
			i.logger.Info().Msg("One minute passed")
		}
	}
}

// HandleNewBlock handles the receive of new block from the chain.
func (i *Indexer) HandleNewBlock(ctx context.Context, blk *tmtypes.Block) error {
	i.b.SetChainHeader(blk)
	i.logger.Info().Int64("height", blk.Height).Msg("new block received")

	for _, tx := range blk.Data.Txs {
		if err := i.HandleTx(ctx, tx); err != nil {
			return err
		}
	}

	return nil
}

// HandleTx handles the receive of new Tx from the chain.
func (i *Indexer) HandleTx(ctx context.Context, tmTx tmtypes.Tx) error {
	tx, err := i.b.DecodeTx(tmTx)
	if err != nil {
		i.logger.Err(err).Msg("error decoding Tx")
		return err
	}

	for _, msg := range tx.GetMsgs() {
		if err := i.HandleMsg(ctx, msg); err != nil {
			i.logger.Err(err).Msg("error handling msg")
			continue
		}
	}
	return nil
}

// HandleMsg handles the receive of new msg from the chain Tx.
func (i *Indexer) HandleMsg(ctx context.Context, msg proto.Message) error {
	// parses to the expected messages.
	msgLiq, ok := msg.(*lvgtypes.MsgLiquidate)
	if ok {
		fmt.Printf("\n is msg liquidate %+v", msg.String())
		fmt.Printf("\n msgLiq%+v", msgLiq)
		return nil
	}

	t, ok := msg.(*oracletypes.MsgAggregateExchangeRatePrevote)
	if ok {
		fmt.Printf("\n MsgAggregateExchangeRatePrevote%+v", t)
		return nil
	}
	fmt.Printf("\n is NOT msg liquidate")

	return nil
}

// onStart loads the starter data into blockchain.
func (i *Indexer) onStart(ctx context.Context) error {
	if err := i.loadChainHeader(ctx); err != nil {
		return err
	}
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
	return i.db.UpsertChainHeader(ctx, chainID, int(height))
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

type CosmosMsgIndexed struct {
	LastBlockHeightIndexed int64
	MsgType                string
}

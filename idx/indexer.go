package idx

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/umee-network/umeed-indexer/database"
)

// Indexer struct responsible for calling blockchain rpc/websocket for data and
// storing that into the database.
type Indexer struct {
	b  Blockchain
	db database.Database
}

// Blockchain is the expected blockchain interface the indexer needs to store data in the database.
type Blockchain interface {
	ChainID() string
	ChainHeader() (chainID string, height uint64, err error)
	SetChainHeader(blk *tmtypes.Block)
	SubscribeEvents(ctx context.Context) (outNewEvt <-chan ctypes.ResultEvent, err error)
	SubscribeNewBlock(ctx context.Context) (outNewBlock <-chan ctypes.ResultEvent, err error)
	Close(ctx context.Context) error
}

// NewIndexer returns a new indexer struct with open connections.
func NewIndexer(ctx context.Context, b Blockchain, db database.Database) (*Indexer, error) {
	i := &Indexer{
		b:  b,
		db: db,
	}
	return i, i.onStart(ctx)
}

// Index starts to index transactions.
func (i *Indexer) Index(ctx context.Context) error {
	evts, err := i.b.SubscribeEvents(ctx)
	if err != nil {
		return err
	}
	newBlock, err := i.b.SubscribeNewBlock(ctx)
	if err != nil {
		return err
	}

	return i.IndexCases(ctx, newBlock, evts)
}

// IndexCases handle all the cases for the indexer.
func (i *Indexer) IndexCases(
	ctx context.Context,
	newBlock, evts <-chan ctypes.ResultEvent,
) error {
	oneMin := time.NewTicker(time.Minute)
	defer oneMin.Stop()

	for {
		select {
		// only closes the connections if the context is done.
		case <-ctx.Done():
			return i.Close(ctx)

		case blk := <-newBlock: // listen to new blocks being produced.
			evtBlock, ok := blk.Data.(tmtypes.EventDataNewBlock)
			if !ok {
				continue
			}
			if err := i.HandleNewBlock(ctx, evtBlock.Block); err != nil {
				fmt.Printf("\nerror on handling new block %s", err.Error())
			}

		case <-oneMin.C: // every minute.
			// store tickers data into database.
			fmt.Printf("\nOne minute has passed")

		case evt := <-evts: // at each new event listened.
			// verifies if it is an expected event and if it is, parses it
			// and store in the database.
			// this handler populates the channel for new swap events (chSwap).
			if err := i.HandleEvt(ctx, evt); err != nil {
				fmt.Printf("\nerror handling evt %s - %+v", err.Error(), evt)
			}
		}
	}
}

// HandleNewBlock handles the receive of new block from the chain.
func (i *Indexer) HandleNewBlock(ctx context.Context, blk *tmtypes.Block) error {
	i.b.SetChainHeader(blk)
	fmt.Printf("\nnew block height %d", blk.Height)

	return nil
}

// HandleEvt iterates over modified pools updating it on the store.
func (i *Indexer) HandleEvt(ctx context.Context, evt ctypes.ResultEvent) error {
	// fmt.Printf("\n event received query: %+s", evt.Query)
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

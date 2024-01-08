package idx

import (
	"context"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

// Blockchain is the expected blockchain interface the indexer needs to store data in the database.
type Blockchain interface {
	ChainID() string
	ChainHeader() (chainID string, height uint64, err error)
	SetChainHeader(blk *tmtypes.Block)
	SubscribeEvents(ctx context.Context) (outNewEvt <-chan ctypes.ResultEvent, err error)
	SubscribeNewBlock(ctx context.Context) (outNewBlock <-chan ctypes.ResultEvent, err error)
	Close(ctx context.Context) error
	DecodeTx(tx tmtypes.Tx) (sdktypes.Tx, error)
}

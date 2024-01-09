package idx

import (
	"context"

	tmtypes "github.com/cometbft/cometbft/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
)

// Blockchain is the expected blockchain interface the indexer needs to store data in the database.
type Blockchain interface {
	Close(ctx context.Context) error
	ChainID() string
	ChainHeader() (chainID string, height uint64, err error)
	SetChainHeader(blk *tmtypes.Block)
	DecodeTx(tx tmtypes.Tx) (sdktypes.Tx, error)
	SubscribeNewBlock(ctx context.Context) (cNewBlock <-chan *tmtypes.Block, err error)
	Block(ctx context.Context, height int64) (blk *tmtypes.Block, minimumBlkHeight int, err error)
}

package idx

import (
	"sync"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/umee-network/umeed-indexer/graph/types"
)

// SafeChainInfo thread safe chain info wrapper.
type SafeChainInfo struct {
	mu sync.Mutex
	*types.ChainInfo
}

// NewSafeChainInfo returns a chainInfo safe structure.
func NewSafeChainInfo(info *types.ChainInfo) *SafeChainInfo {
	return &SafeChainInfo{
		ChainInfo: info,
	}
}

// Update updates the entire chain info structure safely.
func (s *SafeChainInfo) Update(info *types.ChainInfo) {
	_ = s.Execute(func(info *types.ChainInfo) error {
		s.ChainInfo = info
		return nil
	})
}

// Execute executes something with the safe chain info.
func (s *SafeChainInfo) Execute(f func(info *types.ChainInfo) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return f(s.ChainInfo)
}

// Copy returns a copy of cosmos msgs indexed.
func (s *SafeChainInfo) Copy() (cosmosMsgs []*types.CosmosMsgIndexed, lastBlockHeightReceived int) {
	_ = s.Execute(func(info *types.ChainInfo) error {
		cosmosMsgs = make([]*types.CosmosMsgIndexed, len(info.CosmosMsgs))
		copy(cosmosMsgs, info.CosmosMsgs)

		lastBlockHeightReceived = info.LastBlockHeightReceived
		return nil
	})
	return cosmosMsgs, lastBlockHeightReceived
}

// UpdateFromBlock updates the general block info in the chain info.
func (s *SafeChainInfo) UpdateFromBlock(blk *tmtypes.Block) {
	_ = s.Execute(func(info *types.ChainInfo) error {
		s.ChainInfo.LastBlockHeightReceived = int(blk.Height)
		s.ChainInfo.LastBlockTimeUnixReceived = int(blk.Time.Unix())
		s.ChainInfo.ChainID = blk.ChainID
		return nil
	})
}

// IndexBlockForMsg index the block height for an specific msg.
func (s *SafeChainInfo) IndexBlockForMsg(msgName string, blkHeight int) {
	_ = s.Execute(func(info *types.ChainInfo) error {
		s.ChainInfo.IndexBlockHeightForMsg(msgName, blkHeight)
		return nil
	})
}

// IndexBlock index the block height for all msgs.
func (s *SafeChainInfo) IndexBlock(blkHeight int) {
	_ = s.Execute(func(info *types.ChainInfo) error {
		s.ChainInfo.IndexBlockHeight(blkHeight)
		return nil
	})
}

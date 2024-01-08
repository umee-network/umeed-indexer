package types

import (
	"strings"

	"github.com/cosmos/gogoproto/proto"
	lvgtypes "github.com/umee-network/umee/v6/x/leverage/types"
)

var (
	defaultCosmosMsgs []*CosmosMsgIndexed = []*CosmosMsgIndexed{
		{
			ProtoMsgName:           proto.MessageName(&lvgtypes.MsgLiquidate{}),
			BlocksIndexed:          []*BlockIndexedInterval{},
			IdxHeighestBlockHeight: 0,
		},
	}
)

// DefaultChainInfo returns the default chain info.
func DefaultChainInfo(chainID string) *ChainInfo {
	return &ChainInfo{
		ChainID:                 chainID,
		CosmosMsgs:              defaultCosmosMsgs,
		LastBlockHeightReceived: 0,
	}
}

// MergeCosmosMsgIndexedWithDefaults merge the given cosmos msgs with the default ones.
// usefull for when new txs are being indexed and we just need to add that to the default cosmos msg.
func MergeCosmosMsgIndexedWithDefaults(msgs ...*CosmosMsgIndexed) []*CosmosMsgIndexed {
	cosmosMsgs := msgs

	for _, dftMsg := range defaultCosmosMsgs { // 1, 2, 3
		contains := false
		for _, msg := range msgs { // 4,2,5
			if !strings.EqualFold(msg.ProtoMsgName, dftMsg.ProtoMsgName) {
				continue
			}
			contains = true
			break
		}

		if contains {
			continue
		}

		cosmosMsgs = append(cosmosMsgs, dftMsg)
	}

	return cosmosMsgs
}

// MergeWithDefaults loads the defaults of cosmos msgs indexed.
func (c *ChainInfo) MergeWithDefaults() {
	c.CosmosMsgs = MergeCosmosMsgIndexedWithDefaults(c.CosmosMsgs...)
}

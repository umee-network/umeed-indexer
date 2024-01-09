package types

import (
	"sort"
	"strings"

	"github.com/cosmos/gogoproto/proto"
	lvgtypes "github.com/umee-network/umee/v6/x/leverage/types"
)

var (
	MsgNameLiquidate                      = proto.MessageName(&lvgtypes.MsgLiquidate{})
	defaultCosmosMsgs []*CosmosMsgIndexed = []*CosmosMsgIndexed{
		{
			ProtoMsgName:  MsgNameLiquidate,
			BlocksIndexed: []*BlockIndexedInterval{},
		},
	}
	_ sort.Interface = BlockIndexedIntervalSorter{}
)

type BlockIndexedIntervalSorter []*BlockIndexedInterval

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

// NeedsToIndex returns true if the given block height needs to be indexed.
func (c *ChainInfo) NeedsToIndex(blockHeight int) bool {
	for _, cosmosMsg := range c.CosmosMsgs {
		if BlockAlreadyIndexed(blockHeight, cosmosMsg.BlocksIndexed) {
			continue
		}
		return true
	}
	return false
}

// returns true if this block was already indexed.
func BlockAlreadyIndexed(blockHeight int, blocksIndexed []*BlockIndexedInterval) bool {
	for _, blkIndexed := range blocksIndexed {
		// in range
		if blockHeight >= blkIndexed.IdxFromBlockHeight && blockHeight <= blkIndexed.IdxToBlockHeight {
			return true
		}
	}

	return false
}

// LowestBlockHeightToIndex returns the least block height it should start to try to index.
func (c *ChainInfo) LowestBlockHeightToIndex(minHeight int) (blockHeight int) {
	return LowestBlockHeightToIndex(c.CosmosMsgs, minHeight)
}

// BlockHeightIndexed updates the internal values of cosmos msgs of the current block indexed.
func (c *ChainInfo) BlockHeightIndexed(blkHeight int) {

	for _, cosmosMsg := range c.CosmosMsgs {
		if BlockAlreadyIndexed(blkHeight, cosmosMsg.BlocksIndexed) {
			continue
		}

		// from: 5 to: 10
		// from: 14 to 19
		// blk: 13

		// from: 5 to: 10
		// from: 14 to 19
		// blk: 11

		sort.Sort(BlockIndexedIntervalSorter(cosmosMsg.BlocksIndexed))
		// needs to be sorted.
		cosmosMsg.BlocksIndexed = IndexBlockHeightToInterval(cosmosMsg.BlocksIndexed, blkHeight)
	}
}

// IndexBlockHeightToInterval removes the index from the slice.
func IndexBlockHeightToInterval(slice []*BlockIndexedInterval, blkHeightToAdd int) []*BlockIndexedInterval {
	for i, blkIndexed := range slice {
		from, to := blkIndexed.IdxFromBlockHeight, blkIndexed.IdxToBlockHeight

		if blkHeightToAdd == to+1 {
			isLastOnSlice := i+1 == len(slice)

			if isLastOnSlice {
				// it is just increasing one block from the last interval
				blkIndexed.IdxToBlockHeight = blkHeightToAdd
				return slice
			}
			nextIndex := i + 1
			// not the last index on array, it has next interval
			next := slice[nextIndex]
			if next.IdxFromBlockHeight == blkHeightToAdd {
				// should merge two intervals into one.
				blkIndexed.IdxToBlockHeight = next.IdxToBlockHeight
				return RemoveFromBlockIndexedInterval(slice, nextIndex)
			}

			// if the next item is not going to join interval, just increase the current blkIndexed
			blkIndexed.IdxToBlockHeight = blkHeightToAdd
			return slice
		}

		if blkHeightToAdd == from-1 {

			isFirstOnSlice := i == 0
			if isFirstOnSlice {
				// just include the new indexed block into the first interval
				blkIndexed.IdxFromBlockHeight = blkHeightToAdd
				break
			}

			prevIndex := i - 1
			prev := slice[prevIndex]
			if prev.IdxToBlockHeight == blkHeightToAdd {
				// should merge two intervals into one.
				blkIndexed.IdxFromBlockHeight = prev.IdxFromBlockHeight
				return RemoveFromBlockIndexedInterval(slice, prevIndex)
			}

			// if the prev item is not going to join interval, just decrease the current blkIndexed
			blkIndexed.IdxFromBlockHeight = blkHeightToAdd
			return slice
		}
	}

	// if there is no interval neighbor (with one block heigh diff)
	// include a new interval into the slice
	idx := sort.Search(len(slice), func(i int) bool { return slice[i].IdxToBlockHeight < blkHeightToAdd })
	itemToAdd := &BlockIndexedInterval{IdxFromBlockHeight: blkHeightToAdd, IdxToBlockHeight: blkHeightToAdd}
	if idx == len(slice) {
		return append(slice, itemToAdd)
	}

	// insert sorted.
	slice = append(slice[:idx+1], slice[idx:]...)
	// Insert the new element.
	slice[idx] = itemToAdd
	return slice
}

// RemoveFromBlockIndexedInterval removes the index from the slice.
func RemoveFromBlockIndexedInterval(slice []*BlockIndexedInterval, idxToRemove int) []*BlockIndexedInterval {
	return append(slice[:idxToRemove], slice[idxToRemove+1:]...)
}

// LowestBlockHeightToIndex returns the least block height it should start to try to index based on the cosmos msgs already indexed
func LowestBlockHeightToIndex(cosmosMsgs []*CosmosMsgIndexed, minHeight int) (blockHeight int) {
	if len(cosmosMsgs) == 0 { // no need to sort / iterate if there is no msgs to index
		return minHeight
	}

	for _, cosmosMsg := range cosmosMsgs {
		for _, blockIdxed := range cosmosMsg.BlocksIndexed {
			nextBlockToIndex := blockIdxed.IdxToBlockHeight + 1

			if nextBlockToIndex < minHeight {
				continue
			}
			if nextBlockToIndex < blockHeight {
				continue
			}
			blockHeight = nextBlockToIndex
		}
	}

	return max(blockHeight, minHeight)
}

// Len implements sort.Interface.
func (s BlockIndexedIntervalSorter) Len() int {
	return len(s)
}

// Less implements sort.Interface.
func (s BlockIndexedIntervalSorter) Less(i int, j int) bool {
	return s[i].IdxFromBlockHeight < s[j].IdxFromBlockHeight
}

// Swap implements sort.Interface.
func (s BlockIndexedIntervalSorter) Swap(i int, j int) {
	s[i], s[j] = s[j], s[i]
}

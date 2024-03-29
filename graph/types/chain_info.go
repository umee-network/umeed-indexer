package types

import (
	"sort"
	"strings"

	"github.com/cosmos/gogoproto/proto"
	lvgtypes "github.com/umee-network/umee/v6/x/leverage/types"
)

var (
	MsgNameLiquidate                              = proto.MessageName(&lvgtypes.MsgLiquidate{})
	MsgNameLeveragedLiquidate                     = proto.MessageName(&lvgtypes.MsgLeveragedLiquidate{})
	defaultCosmosMsgs         []*CosmosMsgIndexed = []*CosmosMsgIndexed{
		{
			ProtoMsgName:  MsgNameLiquidate,
			BlocksIndexed: []*BlockIndexedInterval{},
		},
		{
			ProtoMsgName:  MsgNameLeveragedLiquidate,
			BlocksIndexed: []*BlockIndexedInterval{},
		},
	}
	_ sort.Interface = BlockIndexedIntervalSorter{}
)

type BlockIndexedIntervalSorter []*BlockIndexedInterval

// DefaultChainInfo returns the default chain info.
func DefaultChainInfo(chainID string) *ChainInfo {
	return &ChainInfo{
		ChainID:                   chainID,
		CosmosMsgs:                defaultCosmosMsgs,
		LastBlockHeightReceived:   0,
		LastBlockTimeUnixReceived: 0,
	}
}

// MergeWithDefault merge with the default of chain info if needed.
func (c *ChainInfo) MergeWithDefault() {
	if len(c.CosmosMsgs) == 0 {
		c.CosmosMsgs = defaultCosmosMsgs
	}

	for _, dftCosmoMsg := range defaultCosmosMsgs {
		defaultExist := false
		for _, cosmoMsg := range c.CosmosMsgs {
			if !strings.EqualFold(cosmoMsg.ProtoMsgName, dftCosmoMsg.ProtoMsgName) {
				continue
			}
			defaultExist = true
			break
		}

		if defaultExist {
			continue
		}
		c.CosmosMsgs = append(c.CosmosMsgs, dftCosmoMsg)
	}
}

// MergeCosmosMsgIndexedWithDefaults merge the given cosmos msgs with the default ones.
// usefull for when new txs are being indexed and we just need to add that to the default cosmos msg.
func MergeCosmosMsgIndexedWithDefaults(msgs ...*CosmosMsgIndexed) []*CosmosMsgIndexed {
	cosmosMsgs := msgs

	for _, dftMsg := range defaultCosmosMsgs {
		contains := false
		for _, msg := range msgs {
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
	return NeedsToIndex(c.CosmosMsgs, blockHeight)
}

// NeedsToIndex returns true if the given block height needs to be indexed.
func NeedsToIndex(cosmosMsgs []*CosmosMsgIndexed, blockHeight int) bool {
	for _, cosmosMsg := range cosmosMsgs {
		if BlockAlreadyIndexed(blockHeight, cosmosMsg.BlocksIndexed) {
			continue
		}
		return true
	}
	return false
}

// NeedsToIndexForMsg returns true if the given block height needs to be indexed.
func NeedsToIndexForMsg(protoMsgToIndex string, cosmosMsgs []*CosmosMsgIndexed, blockHeight int) bool {
	for _, cosmosMsg := range cosmosMsgs {
		if !strings.EqualFold(cosmosMsg.ProtoMsgName, protoMsgToIndex) {
			continue
		}
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

// IndexBlockHeight updates the internal values of cosmos msgs of the current block indexed.
func (c *ChainInfo) IndexBlockHeight(blkHeight int) {

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

		cosmosMsg.BlocksIndexed = IndexBlockHeightToInterval(cosmosMsg.BlocksIndexed, blkHeight)
	}
}

// IndexBlockHeightForMsg updates the internal values of cosmos msgs of the current block indexed.
func (c *ChainInfo) IndexBlockHeightForMsg(msgName string, blkHeight int) (indexed bool) {
	// TODO: add tests.
	for _, cosmosMsg := range c.CosmosMsgs {
		if !strings.EqualFold(msgName, cosmosMsg.ProtoMsgName) {
			continue
		}
		if BlockAlreadyIndexed(blkHeight, cosmosMsg.BlocksIndexed) {
			continue
		}

		cosmosMsg.BlocksIndexed = IndexBlockHeightToInterval(cosmosMsg.BlocksIndexed, blkHeight)
		return true
	}
	return false
}

// TODO: add test to this.
// IndexBlockHeightToInterval removes the index from the slice.
func IndexBlockHeightToInterval(slice []*BlockIndexedInterval, heightToAdd int) []*BlockIndexedInterval {
	itemToAdd := &BlockIndexedInterval{IdxFromBlockHeight: heightToAdd, IdxToBlockHeight: heightToAdd}

	if len(slice) == 0 {
		return []*BlockIndexedInterval{itemToAdd}
	}

	// needs to be sorted.
	sort.Sort(BlockIndexedIntervalSorter(slice))

	for i, blkIndexed := range slice {
		from, to := blkIndexed.IdxFromBlockHeight, blkIndexed.IdxToBlockHeight
		if heightToAdd >= from && heightToAdd <= to {
			// height was already indexed, return the slice.
			return slice
		}

		isLastOnSlice := i+1 == len(slice)
		nextIndex := i + 1

		if heightToAdd == to+1 {
			if isLastOnSlice {
				// it is just increasing one block from the last interval
				blkIndexed.IdxToBlockHeight = heightToAdd
				return slice
			}
			// not the last index on array, it has next interval
			next := slice[nextIndex]
			if next.IdxFromBlockHeight == heightToAdd+1 {
				// should merge two intervals into one.
				blkIndexed.IdxToBlockHeight = next.IdxToBlockHeight
				return RemoveFromBlockIndexedInterval(slice, nextIndex)
			}

			// if the next item is not going to join interval, just increase the current blkIndexed
			blkIndexed.IdxToBlockHeight = heightToAdd
			return slice
		}

		if heightToAdd == from-1 {

			isFirstOnSlice := i == 0
			if isFirstOnSlice {
				// just include the new indexed block into the first interval
				blkIndexed.IdxFromBlockHeight = heightToAdd
				break
			}

			prevIndex := i - 1
			prev := slice[prevIndex]
			if prev.IdxToBlockHeight == heightToAdd-1 {
				// should merge two intervals into one.
				blkIndexed.IdxFromBlockHeight = prev.IdxFromBlockHeight
				return RemoveFromBlockIndexedInterval(slice, prevIndex)
			}

			// if the prev item is not going to join interval, just decrease the current blkIndexed
			blkIndexed.IdxFromBlockHeight = heightToAdd
			return slice
		}

		if isLastOnSlice { // just break and add to the end
			break
		}

		// since we sorted before, it is ordered and we can check the interval for the next
		next := slice[nextIndex]
		if heightToAdd < next.IdxFromBlockHeight { // insert at that position
			return append(slice[:nextIndex], append([]*BlockIndexedInterval{itemToAdd}, slice[nextIndex:]...)...)
		}

	}

	// not neighbour of any interval, append to the end
	return append(slice, itemToAdd)
	// // if there is no interval neighbor (with one block heigh diff)
	// // include a new interval into the slice

	// idx := sort.Search(len(slice), func(i int) bool { return slice[i].IdxToBlockHeight < heightToAdd })
	// itemToAdd := &BlockIndexedInterval{IdxFromBlockHeight: blkHeightToAdd, IdxToBlockHeight: blkHeightToAdd}

	// if idx+1 == len(slice) {
	// 	return append(slice, itemToAdd)
	// }

	// // insert sorted.
	// slice = append(slice[:idx+1], slice[idx:]...)
	// // Insert the new element.
	// slice[idx] = itemToAdd
	// return slice
}

// RemoveFromBlockIndexedInterval removes the index from the slice.
func RemoveFromBlockIndexedInterval(slice []*BlockIndexedInterval, idxToRemove int) []*BlockIndexedInterval {
	return append(slice[:idxToRemove], slice[idxToRemove+1:]...)
}

// LowestBlockHeightToIndex returns the least block height it should start to try to index based on the cosmos msgs already indexed
func LowestBlockHeightToIndex(cosmosMsgs []*CosmosMsgIndexed, minHeight int) (blockHeight int) {
	blockHeight = 1    // starts at one
	if minHeight < 1 { // makes sure the min height is always 1.
		minHeight = 1
	}
	if len(cosmosMsgs) == 0 { // no need to sort / iterate if there is no msgs to index
		return minHeight
	}

	blockHeightSetByMsg := false
	for _, cosmosMsg := range cosmosMsgs {
		for i, blockIdxed := range cosmosMsg.BlocksIndexed {
			nextBlockToIndex := blockIdxed.IdxToBlockHeight + 1

			if i == 0 && blockIdxed.IdxFromBlockHeight > minHeight {
				// there is a gap in the beggining, like indexer 3 ~ 6 with minHeight as 1
				// it should set as minHeight.
				blockHeightSetByMsg = true
				blockHeight = minHeight
				continue
			}

			if nextBlockToIndex < minHeight {
				// next possible block is lower than the minHeight
				continue
			}
			if !blockHeightSetByMsg && blockIdxed.IdxFromBlockHeight > minHeight { // minHeight was not indexed yet.
				blockHeightSetByMsg = true
				blockHeight = minHeight
				continue
			}
			if blockHeightSetByMsg && nextBlockToIndex > blockHeight {
				// next possible block was already set and it is bigger than the previous set
				continue
			}

			blockHeightSetByMsg = true
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

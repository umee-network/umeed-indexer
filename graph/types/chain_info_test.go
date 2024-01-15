package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/umee-network/umeed-indexer/graph/types"
)

func TestLowestBlockHeightToIndex(t *testing.T) {
	tcs := []struct {
		title      string
		cosmosMsgs []*types.CosmosMsgIndexed
		minHeight  int

		expectedBlockHeight int
	}{
		{
			"zero - empty cosmos, min height 0 = 1",
			[]*types.CosmosMsgIndexed{},
			0,
			1,
		},
		{
			"zero - empty cosmos, min height 15 = 15",
			[]*types.CosmosMsgIndexed{},
			15,
			15,
		},
		{
			"interval 4 ~ 10, min height 0 = 1",
			[]*types.CosmosMsgIndexed{
				msgCosmosLiquidate(4, 10),
			},
			0,
			1,
		},
		{
			"interval 1 ~ 2, 4 ~ 10, min height 0 = 3",
			[]*types.CosmosMsgIndexed{
				msgCosmosLiquidate(1, 2, 4, 10),
			},
			0,
			3,
		},
		{
			"interval 1 ~ 2, 4 ~ 10, min height 4 = 11",
			[]*types.CosmosMsgIndexed{
				msgCosmosLiquidate(1, 2, 4, 10),
			},
			4,
			11,
		},
		{
			"interval 15 ~ 167, 200 ~ 215, 10 ~ 50, 88 ~ 102 min height 100 = 103",
			[]*types.CosmosMsgIndexed{
				msgCosmosLiquidate(15, 167, 200, 215),
				msgCosmosLiquidate(10, 50, 88, 102),
			},
			100,
			103,
		},
		{
			"interval 8713586 ~ 8739758, 10149582 ~ 10153387 min height 9063670 = 9063670",
			[]*types.CosmosMsgIndexed{
				msgCosmosLiquidate(8713586, 8739758, 10149582, 10153387),
			},
			9063670,
			9063670,
		},
		{
			"interval 8713586 ~ 8739758, 10149582 ~ 10153387, 8713586 ~ 9063673, 10149582 ~ 10153387, min height 9063670 = 9063670",
			[]*types.CosmosMsgIndexed{
				msgCosmosLiquidate(8713586, 8739758, 10149582, 10153387),
				msgCosmosLiquidate(8713586, 9063673, 10149582, 10153387),
			},
			9063670,
			9063670,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			actBlockHeight := types.LowestBlockHeightToIndex(tc.cosmosMsgs, tc.minHeight)
			require.Equal(t, tc.expectedBlockHeight, actBlockHeight)
		})
	}
}

func TestIndexBlockHeightToInterval(t *testing.T) {
	tcs := []struct {
		title          string
		intervals      []*types.BlockIndexedInterval
		blkHeightToAdd int

		expected []*types.BlockIndexedInterval
	}{
		{
			"empty, add blk 1 = 1 ~ 1",
			[]*types.BlockIndexedInterval{},
			1,
			blockIntervals(1, 1),
		},
		{
			"empty, add blk 15 = 15 ~ 15",
			[]*types.BlockIndexedInterval{},
			15,
			blockIntervals(15, 15),
		},
		{
			"3~4, add blk 15 = 3 ~ 4, 15 ~ 15",
			blockIntervals(3, 4),
			15,
			blockIntervals(3, 4, 15, 15),
		},
		{
			"3~4, add blk 3 = 3 ~ 4",
			blockIntervals(3, 4),
			3,
			blockIntervals(3, 4),
		},
		{
			"3~4, add blk 5 = 3 ~ 5",
			blockIntervals(3, 4),
			5,
			blockIntervals(3, 5),
		},
		{
			"3~4,7~10 add blk 8 = 3~4,7~10",
			blockIntervals(3, 4, 7, 10),
			8,
			blockIntervals(3, 4, 7, 10),
		},
		{
			"3~4,7~10 add blk 5 = 3~5,7~10",
			blockIntervals(3, 4, 7, 10),
			5,
			blockIntervals(3, 5, 7, 10),
		},
		{
			"3~4,8~10 add blk 6 = 3~4,6~6,8~10",
			blockIntervals(3, 4, 8, 10),
			6,
			blockIntervals(3, 4, 6, 6, 8, 10),
		},
		{
			"3~4,6~6,8~10 add blk 7 = 3~4,6~10",
			blockIntervals(3, 4, 6, 6, 8, 10),
			7,
			blockIntervals(3, 4, 6, 10),
		},
		{
			"3~4,6~6,8~10 add blk 5 = 3~6,8~10",
			blockIntervals(3, 4, 6, 6, 8, 10),
			5,
			blockIntervals(3, 6, 8, 10),
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			act := types.IndexBlockHeightToInterval(tc.intervals, tc.blkHeightToAdd)
			require.Equal(t, tc.expected, act)
		})
	}
}

func msgCosmosLiquidate(fromTos ...int) (msg *types.CosmosMsgIndexed) {
	return msgCosmos(types.MsgNameLiquidate, fromTos...)
}

func msgCosmos(name string, fromTos ...int) (msg *types.CosmosMsgIndexed) {
	return &types.CosmosMsgIndexed{
		ProtoMsgName:  name,
		BlocksIndexed: blockIntervals(fromTos...),
	}
}

func blockIntervals(fromTos ...int) (intervals []*types.BlockIndexedInterval) {
	intervals = make([]*types.BlockIndexedInterval, 0, len(fromTos)/2)
	for i := 0; i < len(fromTos); i += 2 {
		intervals = append(intervals, &types.BlockIndexedInterval{
			IdxFromBlockHeight: fromTos[i],
			IdxToBlockHeight:   fromTos[i+1],
		})
	}
	return intervals
}

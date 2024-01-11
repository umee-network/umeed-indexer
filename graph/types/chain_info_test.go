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
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			actBlockHeight := types.LowestBlockHeightToIndex(tc.cosmosMsgs, tc.minHeight)
			require.Equal(t, tc.expectedBlockHeight, actBlockHeight)
		})
	}
}

func msgCosmosLiquidate(fromTos ...int) (msg *types.CosmosMsgIndexed) {
	return msgCosmos(types.MsgNameLiquidate, fromTos...)
}

func msgCosmos(name string, fromTos ...int) (msg *types.CosmosMsgIndexed) {
	msg = &types.CosmosMsgIndexed{
		ProtoMsgName:  name,
		BlocksIndexed: make([]*types.BlockIndexedInterval, 0, len(fromTos)/2),
	}

	for i := 0; i < len(fromTos); i += 2 {
		msg.BlocksIndexed = append(msg.BlocksIndexed, &types.BlockIndexedInterval{
			IdxFromBlockHeight: fromTos[i],
			IdxToBlockHeight:   fromTos[i+1],
		})
	}

	return msg
}

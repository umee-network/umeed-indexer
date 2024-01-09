package types

import (
	lvgtypes "github.com/umee-network/umee/v6/x/leverage/types"
)

// ParseTxLeverageLiquidate gets an lvg tx msg and transpile to the graphql one.
func ParseTxLeverageLiquidate(lvgMsg *lvgtypes.MsgLiquidate) MsgLiquidate {
	return MsgLiquidate{
		Liquidator:  lvgMsg.Liquidator,
		Borrower:    lvgMsg.Borrower,
		Repayment:   lvgMsg.Repayment.String(),
		RewardDenom: lvgMsg.RewardDenom,
	}
}

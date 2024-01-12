package types

import (
	lvgtypes "github.com/umee-network/umee/v6/x/leverage/types"
)

// ParseTxLiquidate gets an lvg tx msg and transpile to the graphql one.
func ParseTxLiquidate(lvgMsg *lvgtypes.MsgLiquidate) MsgLiquidate {
	return MsgLiquidate{
		Liquidator:  lvgMsg.Liquidator,
		Borrower:    lvgMsg.Borrower,
		Repayment:   lvgMsg.Repayment.String(),
		RewardDenom: lvgMsg.RewardDenom,
	}
}

// ParseTxLeverageLiquidate gets an lvg tx msg and transpile to the graphql one.
func ParseTxLeverageLiquidate(lvgMsg *lvgtypes.MsgLeveragedLiquidate) MsgLeverageLiquidate {
	return MsgLeverageLiquidate{
		Liquidator:  lvgMsg.Liquidator,
		Borrower:    lvgMsg.Borrower,
		RepayDenom:  lvgMsg.RepayDenom,
		RewardDenom: lvgMsg.RewardDenom,
		MaxRepay:    lvgMsg.MaxRepay.String(),
	}
}

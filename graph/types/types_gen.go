// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package types

type BlockIndexedInterval struct {
	IdxFromBlockHeight int `json:"idxFromBlockHeight" firestore:"idxFromBlockHeight"`
	IdxToBlockHeight   int `json:"idxToBlockHeight" firestore:"idxToBlockHeight"`
}

type ChainInfo struct {
	LastBlockHeightReceived int                 `json:"lastBlockHeightReceived" firestore:"lastBlockHeightReceived"`
	ChainID                 string              `json:"chainID" firestore:"chainID"`
	CosmosMsgs              []*CosmosMsgIndexed `json:"cosmosMsgs" firestore:"cosmosMsgs"`
}

type CosmosMsgIndexed struct {
	ProtoMsgName           string                  `json:"protoMsgName" firestore:"protoMsgName"`
	BlocksIndexed          []*BlockIndexedInterval `json:"blocksIndexed" firestore:"blocksIndexed"`
	IdxHeighestBlockHeight int                     `json:"idxHeighestBlockHeight" firestore:"idxHeighestBlockHeight"`
}

type IndexedTx struct {
	TxHash       string        `json:"txHash" firestore:"txHash"`
	ProtoMsgName string        `json:"protoMsgName" firestore:"protoMsgName"`
	BlockHeight  int           `json:"blockHeight" firestore:"blockHeight"`
	MsgLiquidate *MsgLiquidate `json:"msgLiquidate,omitempty" firestore:"msgLiquidate"`
}

type MsgLiquidate struct {
	Liquidator  string `json:"liquidator" firestore:"liquidator"`
	Borrower    string `json:"borrower" firestore:"borrower"`
	Repayment   string `json:"repayment" firestore:"repayment"`
	RewardDenom string `json:"rewardDenom" firestore:"rewardDenom"`
}

type Query struct {
}

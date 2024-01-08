package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	cmtquery "github.com/cometbft/cometbft/libs/pubsub/query"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	types "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/query"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	umeeapp "github.com/umee-network/umee/v6/app"
	umeeparams "github.com/umee-network/umee/v6/app/params"
)

const (
	// ignoredField tendermint has parameters that are not being used.
	ignoredField = "ignored-field"
	// default pagination limit
	defaultLimit = 10000
)

// Blockchain defines the structure to get information about the chain.
type Blockchain struct {
	conn *Conn

	muRpcID   sync.Mutex
	rpcRespID uint32

	chainID string

	qBank              banktypes.QueryClient
	umeeEncodingConfig testutil.TestEncodingConfig
}

// NewBlockchain returns a new blockchain structure with a RPC connection
// stablish, it errors out if the connection is not setup properly.
// rpcEndpoint ex.: tcp://0.0.0.0:26657, https://umee-rpc.polkachu.com:443
// grpcEndpoint ex.: 127.0.0.1:9090.
func NewBlockchain(rpc, grpc string) (*Blockchain, error) {
	conn, err := NewConn(rpc, grpc)
	if err != nil {
		return nil, err
	}

	if err := conn.Start(); err != nil {
		return nil, err
	}

	encodingConfig := umeeparams.MakeEncodingConfig(umeeModBasics()...)

	return &Blockchain{
		conn:               conn,
		rpcRespID:          0,
		chainID:            "",
		qBank:              banktypes.NewQueryClient(conn.grpcConn),
		umeeEncodingConfig: encodingConfig,
	}, nil
}

func umeeModBasics() (modules []module.AppModuleBasic) {
	umeeBasicManager := umeeapp.ModuleBasics
	modules = make([]module.AppModuleBasic, 0, len(umeeBasicManager))
	for _, v := range umeeBasicManager {
		modules = append(modules, v)
	}
	return modules
}

// DenomsMetadata queries the chain and returns all the denoms metadata available.
func (b *Blockchain) DenomsMetadata(ctx context.Context) (denomsMetadata []banktypes.Metadata, err error) {
	resp, err := b.qBank.DenomsMetadata(ctx, &banktypes.QueryDenomsMetadataRequest{
		Pagination: defaultPaginationRequest(),
	})
	if err != nil {
		return nil, fmt.Errorf("error querying denoms metadata: %w", err)
	}
	return resp.Metadatas, nil
}

// DenomMetadata queries the chain and returns the metadata from that denom.
func (b *Blockchain) DenomMetadata(ctx context.Context, denom string) (denomMetadata banktypes.Metadata, err error) {
	resp, err := b.qBank.DenomMetadata(ctx, &banktypes.QueryDenomMetadataRequest{
		Denom: denom,
	})
	if err != nil {
		return denomMetadata, fmt.Errorf("error querying denom: %s - metadata: %w", denom, err)
	}
	return resp.Metadata, nil
}

// SubscribeEvents subscribe to all the events of tendermint Tx.
func (b *Blockchain) SubscribeEvents(ctx context.Context) (outNewEvt <-chan ctypes.ResultEvent, err error) {
	queryStr := fmt.Sprintf(
		"%s='%s'",
		tmtypes.EventTypeKey, tmtypes.EventTx,
	)

	return b.conn.websocketRPC.Subscribe(ctx, ignoredField, cmtquery.MustParse(queryStr).String())
}

// SubscribeNewBlock subscribe to every new block.
func (b *Blockchain) SubscribeNewBlock(ctx context.Context) (outNewBlock <-chan ctypes.ResultEvent, err error) {
	// chanResultEvtNewBlock, err := b.conn.websocketRPC.Subscribe(ctx, ignoredField, tmtypes.EventQueryNewBlock.String())
	// if err != nil {
	// 	return nil, err
	// }

	// for {
	// 	select {
	// 	// only closes the connections if the context is done.
	// 	case <-ctx.Done():
	// 	case blk := <-chanResultEvtNewBlock: // listen to new blocks being produced.
	// 		evtNewBlock, ok := blk.Data.(tmtypes.EventDataNewBlock)
	// 		if !ok {
	// 			continue
	// 		}
	// 	}
	// }

	return b.conn.websocketRPC.Subscribe(ctx, ignoredField, tmtypes.EventQueryNewBlock.String())
}

// JSONRPCID returns a value for the JSON RPC ID.
// Used to control responses from the rpc query.
func (b *Blockchain) JSONRPCID() uint32 {
	b.muRpcID.Lock()
	defer b.muRpcID.Unlock()
	b.rpcRespID++
	return b.rpcRespID
}

// ChainID returns the chainID. If it doesn't have it stored
// it queries the chain and loads it into the struct
func (b *Blockchain) ChainID() string {
	return b.chainID
}

// Close closes all the open connections the blockchain might have.
func (b *Blockchain) Close(ctx context.Context) error {
	return b.conn.Close(ctx)
}

// SetChainHeader updates the data inside the blockchain as needed.
func (b *Blockchain) SetChainHeader(blk *tmtypes.Block) {
	b.chainID = blk.ChainID
}

// DecodeTx decodes a tx into msgs.
func (b *Blockchain) DecodeTx(tx tmtypes.Tx) (sdktypes.Tx, error) {
	return b.umeeEncodingConfig.TxConfig.TxDecoder()(tx)
}

// ChainHeader queries the chain by the last block height.
func (b *Blockchain) ChainHeader() (string, uint64, error) {
	idSent := types.JSONRPCIntID(b.JSONRPCID())
	req := types.NewRPCRequest(idSent, "block", nil)

	var respRPC RPCRespChainID
	if err := b.makeRPCRequest(req, &respRPC); err != nil {
		return "", 0, err
	}

	height, err := strconv.ParseUint(respRPC.Result.Block.Header.Height, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("error parsing height: %w", err)
	}

	b.chainID = respRPC.Result.Block.Header.ChainID
	return b.chainID, height, nil
}

// makeRPCRequest sends an RPC request to the blockchain and decodes the response.
func (b *Blockchain) makeRPCRequest(req any, responseStruct any) error {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshalling request: %w", err)
	}

	resp, err := b.conn.httpClient.Post(b.conn.AddrRPC, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return fmt.Errorf("error making RPC request: %w", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(responseStruct); err != nil {
		return fmt.Errorf("error decoding RPC response: %w", err)
	}
	return nil
}

// defaultPaginationRequest provides a default pagination request for querying the network.
func defaultPaginationRequest() *query.PageRequest {
	return &query.PageRequest{
		Limit: defaultLimit,
	}
}

// Block returns the block for that given height
func (b *Blockchain) Block(ctx context.Context, height int64) (*tmtypes.Block, error) {
	blkResult, err := b.conn.websocketRPC.Block(ctx, &height)
	if err != nil {
		return nil, err
	}
	return blkResult.Block, nil
}

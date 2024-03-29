package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	types "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	umeeapp "github.com/umee-network/umee/v6/app"
	umeeparams "github.com/umee-network/umee/v6/app/params"
)

const (
	// ignoredField tendermint has parameters that are not being used.
	ignoredField = "ignored-field"
)

// Blockchain defines the structure to get information about the chain.
type Blockchain struct {
	mu        sync.Mutex
	conn      *Conn
	rpcRespID uint32

	chainID string

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

// SubscribeNewBlock subscribe to every new block.
func (b *Blockchain) SubscribeNewBlock(ctx context.Context) (cNewBlock <-chan *tmtypes.Block, err error) {
	chanResultEvtNewBlock, err := b.conn.websocketRPC.Subscribe(ctx, ignoredField, tmtypes.EventQueryNewBlock.String())
	if err != nil {
		return nil, err
	}

	channelNewBlock := make(chan *tmtypes.Block, 1)

	go func() {
		for {
			select {
			// only closes the connections if the context is done.
			case <-ctx.Done():
			case blk := <-chanResultEvtNewBlock: // listen to new blocks being produced.
				evtNewBlock, ok := blk.Data.(tmtypes.EventDataNewBlock)
				if !ok {
					continue
				}
				channelNewBlock <- evtNewBlock.Block
			}
		}
	}()

	return channelNewBlock, nil
}

// JSONRPCID returns a value for the JSON RPC ID.
// Used to control responses from the rpc query.
func (b *Blockchain) JSONRPCID() uint32 {
	b.mu.Lock()
	defer b.mu.Unlock()
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

// Block returns the block for that given height
func (b *Blockchain) Block(ctx context.Context, height int64) (blk *tmtypes.Block, minimumBlkHeight int, err error) {
	// it pannics inside cometBFT if the mutex is not used.
	b.mu.Lock()
	defer b.mu.Unlock()
	blkResult, err := b.conn.websocketRPC.Block(ctx, &height)
	if err != nil {
		// usually a node does not have all the blocks, in this case we could parse the last block that node has available and start from there.
		// "error in json rpc client, with http response metadata: (Status: 200 OK, Protocol HTTP/1.1). RPC error -32603 - Internal error: height 1 is not available, lowest height is 7942001"
		errString := err.Error()
		searchStrInErr := fmt.Sprintf("Internal error: height %d is not available, lowest height is ", height)
		idx := strings.Index(errString, searchStrInErr)
		if idx == -1 {
			return nil, 0, err
		}
		lowestBlockHeightOnNode := errString[idx+len(searchStrInErr):]
		minimumBlkHeight, convErr := strconv.Atoi(lowestBlockHeightOnNode)
		if convErr != nil {
			return nil, 0, errors.Join(err, convErr)
		}
		return nil, minimumBlkHeight, nil
	}
	blk = blkResult.Block
	return blk, int(blk.Height), nil
}

// CheckTx returns nil if the tx was processed correctly without any errors.
func (b *Blockchain) CheckTx(ctx context.Context, tx tmtypes.Tx) (err error) {
	// it pannics inside cometBFT if the mutex is not used.
	b.mu.Lock()
	defer b.mu.Unlock()

	txResult, err := b.conn.websocketRPC.Tx(ctx, tx.Hash(), true)
	if err != nil {
		return err
	}

	if txResult.TxResult.IsErr() {
		return fmt.Errorf("error checking tx %s - %+v", tx.String(), txResult)
	}

	return nil
}

package chain

import (
	"context"
	"net/http"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	tmjsonclient "github.com/cometbft/cometbft/rpc/jsonrpc/client"
	"github.com/cosmos/cosmos-sdk/codec"
)

// Conn stores the connections needed to index events and transactions.
type Conn struct {
	httpClient   *http.Client
	websocketRPC *rpchttp.HTTP
	grpcConn     *grpc.ClientConn
	AddrRPC      string
}

// NewConn returns a new pointer to the connection structure.
// examples of rpc endpoints are:
// https://umee-rpc.polkachu.com:443
// https://umee-rpc.polkachu.com:26657
// https://umee-rpc.polkachu.com:443
// tcp://0.0.0.0:26657
// examples of grpc endpoints are:
// 127.0.0.1:9090
func NewConn(rpc, grpc string) (*Conn, error) {
	httpClient, err := tmjsonclient.DefaultHTTPClient(rpc)
	if err != nil {
		return nil, err
	}
	httpClient.Timeout = 0

	websocketRPC, err := rpchttp.NewWithClient(rpc, "/websocket", httpClient)
	if err != nil {
		return nil, err
	}

	// Create a connection to the gRPC server.
	grpcConn, err := ggrpc.Dial(
		grpc, // your gRPC server address.
		ggrpc.WithTransportCredentials(insecure.NewCredentials()), // The Cosmos SDK doesn't support any transport security mechanism.
		// This instantiates a general gRPC codec which handles proto bytes. We pass in a nil interface registry
		// if the request/response types contain interface instead of 'nil' you should pass the application specific codec.
		ggrpc.WithDefaultCallOptions(ggrpc.ForceCodec(codec.NewProtoCodec(nil).GRPCCodec())),
	)
	if err != nil {
		return nil, err
	}

	return &Conn{
		httpClient:   httpClient,
		websocketRPC: websocketRPC,
		grpcConn:     grpcConn,
		AddrRPC:      rpc,
	}, nil
}

// Start initiate the websocket to be able to subscribe to events.
func (c *Conn) Start() error {
	if c.websocketRPC.IsRunning() {
		return nil
	}
	return c.websocketRPC.Start()
}

// Close ends the connections open.
func (c *Conn) Close(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return c.websocketRPC.UnsubscribeAll(ctx, ignoredField)
	})
	g.Go(func() error {
		return c.websocketRPC.Stop()
	})
	g.Go(func() error {
		return c.grpcConn.Close()
	})

	return g.Wait()
}

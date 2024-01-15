package server

import (
	"context"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/umee-network/umeed-indexer/database"
	"github.com/umee-network/umeed-indexer/graph"
)

// NewRouter creates a new router wiht a database.
func NewRouter(
	ctx context.Context,
	db database.Database,
	logger zerolog.Logger,
) (r *mux.Router, err error) {
	r = mux.NewRouter()

	// Set up the GraphQL server
	config := graph.Config{Resolvers: graph.NewResolver(db, logger)}
	r.Handle("/graphql", newServer(config))

	return r, nil
}

// newServer returns a new server handler based on the graphql schema.
func newServer(config graph.Config) (srv *handler.Server) {
	srv = handler.New(graph.NewExecutableSchema(config))
	srv.AddTransport(&transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})

	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New(1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})
	return srv
}

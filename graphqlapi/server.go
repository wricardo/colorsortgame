// Package graphqlapi wires up the GraphQL API and the static web UI. It has
// no main of its own — start it via `colorsort serve` (see cmd/colorsort).
package graphqlapi

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/wricardo/colorsortgame/graphqlapi/graph"
)

//go:embed static
var staticFiles embed.FS

// Handler returns an http.Handler for the GraphQL API and static UI.
// Used by both Serve and the CLI's ephemeral server spawning.
func Handler() (http.Handler, error) {
	resolver, err := graph.NewResolver()
	if err != nil {
		return nil, fmt.Errorf("load resolver: %w", err)
	}

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("load static assets: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.Handle("/playground", playground.Handler("GraphQL playground", "/query"))
	mux.Handle("/query", srv)

	return mux, nil
}

// Serve starts the GraphQL API and static UI on the given port, blocking
// until the server stops or fails.
func Serve(port string) error {
	mux, err := Handler()
	if err != nil {
		return err
	}

	log.Printf("UI at http://localhost:%s/, GraphQL playground at http://localhost:%s/playground", port, port)
	return http.ListenAndServe(":"+port, mux)
}

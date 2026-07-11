package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/wricardo/colorsortgame"
	"github.com/wricardo/colorsortgame/graphqlapi"
)

// Client wraps a GraphQL endpoint and manages an optional local server.
type Client struct {
	URL    string
	server *http.Server
	closer func() error
}

// New creates a client. If apiURL is empty, spawns a local ephemeral server.
func New(apiURL string) (*Client, error) {
	if apiURL != "" {
		return &Client{URL: apiURL}, nil
	}

	// Spawn ephemeral server on random port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen on random port: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	mux, err := graphqlapi.Handler()
	if err != nil {
		return nil, fmt.Errorf("create handler: %w", err)
	}

	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}
	go server.ListenAndServe()

	// Wait for server to be ready.
	url := fmt.Sprintf("http://127.0.0.1:%d/query", port)
	for i := 0; i < 10; i++ {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return &Client{
		URL:    url,
		server: server,
		closer: func() error { return server.Close() },
	}, nil
}

// Close shuts down the local server if we spawned one.
func (c *Client) Close() error {
	if c.closer != nil {
		return c.closer()
	}
	return nil
}

// Execute runs a GraphQL query or mutation and returns the full response body.
func (c *Client) Execute(ctx context.Context, query string, vars map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": vars,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to read the response body for more info
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Check for GraphQL errors.
	if errs, ok := result["errors"].([]interface{}); ok && len(errs) > 0 {
		if errObj, ok := errs[0].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok {
				return nil, fmt.Errorf("GraphQL: %s", msg)
			}
		}
		return nil, fmt.Errorf("GraphQL error")
	}

	return result, nil
}

// GraphQL response structures (mirrors graphqlapi models).
type GraphQLGame struct {
	ID         string     `json:"id"`
	LevelID    int        `json:"levelId"`
	Difficulty string     `json:"difficulty"`
	Capacity   int        `json:"capacity"`
	Tubes      [][]string `json:"tubes"`
	Moves      int        `json:"moves"`
	History    []Move     `json:"history"`
	Solved     bool       `json:"solved"`
	Stuck      bool       `json:"stuck"`
}

type Move struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type GraphQLLevel struct {
	ID           int    `json:"id"`
	Difficulty   string `json:"difficulty"`
	TubeCapacity int    `json:"tubeCapacity"`
	Tubes        [][]string `json:"tubes"`
}

type GraphQLSolveResult struct {
	Solvable bool     `json:"solvable"`
	Unknown  bool     `json:"unknown"`
	MinMoves int      `json:"minMoves"`
	Path     []string `json:"path"`
}

// ToSave converts a GraphQL Game to a colorsort.Save for display.
func (g *GraphQLGame) ToSave() *colorsort.Save {
	tubes := make([]colorsort.Tube, len(g.Tubes))
	for i, t := range g.Tubes {
		tubes[i] = colorsort.Tube(t)
	}

	history := make([]colorsort.Move, len(g.History))
	for i, m := range g.History {
		history[i] = colorsort.Move{From: m.From, To: m.To}
	}

	return &colorsort.Save{
		LevelID:  g.LevelID,
		Capacity: g.Capacity,
		Tubes:    tubes,
		Moves:    g.Moves,
		History:  history,
		Solved:   g.Solved,
		Stuck:    g.Stuck,
	}
}

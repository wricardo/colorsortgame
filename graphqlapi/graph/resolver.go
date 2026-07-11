package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	"fmt"
	"sync"

	"github.com/wricardo/colorsortgame"
	"github.com/wricardo/colorsortgame/graphqlapi/graph/model"
)

// Resolver holds the in-memory game store and the embedded level set. Games
// are keyed by a generated id since GraphQL has no equivalent to the CLI's
// save-file path.
type Resolver struct {
	mu     sync.Mutex
	levels *colorsort.LevelsFile
	games  map[string]*colorsort.Save

	subMu sync.Mutex
	subs  map[string]map[chan *model.Game]struct{}
}

func NewResolver() (*Resolver, error) {
	lf, err := colorsort.DefaultLevels()
	if err != nil {
		return nil, fmt.Errorf("load embedded levels: %w", err)
	}
	return &Resolver{
		levels: lf,
		games:  map[string]*colorsort.Save{},
		subs:   map[string]map[chan *model.Game]struct{}{},
	}, nil
}

// subscribe registers a channel for updates to gameID, returning an unsubscribe func.
func (r *Resolver) subscribe(gameID string) (chan *model.Game, func()) {
	ch := make(chan *model.Game, 1)

	r.subMu.Lock()
	if r.subs[gameID] == nil {
		r.subs[gameID] = map[chan *model.Game]struct{}{}
	}
	r.subs[gameID][ch] = struct{}{}
	r.subMu.Unlock()

	return ch, func() {
		r.subMu.Lock()
		delete(r.subs[gameID], ch)
		if len(r.subs[gameID]) == 0 {
			delete(r.subs, gameID)
		}
		r.subMu.Unlock()
		close(ch)
	}
}

// publish notifies all subscribers of gameID with a snapshot of the current
// game state. Must be called with the game's state already settled (the
// caller's r.mu still held is fine, since this only touches subMu).
// Non-blocking: a stale pending update for a slow subscriber is dropped in
// favor of the latest one rather than blocking the mutation caller.
func (r *Resolver) publish(gameID string, s *colorsort.Save) {
	game := r.toModelGame(gameID, s)

	r.subMu.Lock()
	defer r.subMu.Unlock()
	for ch := range r.subs[gameID] {
		select {
		case ch <- game:
		default:
			select {
			case <-ch:
			default:
			}
			ch <- game
		}
	}
}

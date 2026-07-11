package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

import (
	"fmt"
	"sync"

	"github.com/wricardo/colorsortgame"
)

// Resolver holds the in-memory game store and the embedded level set. Games
// are keyed by a generated id since GraphQL has no equivalent to the CLI's
// save-file path.
type Resolver struct {
	mu     sync.Mutex
	levels *colorsort.LevelsFile
	games  map[string]*colorsort.Save
}

func NewResolver() (*Resolver, error) {
	lf, err := colorsort.DefaultLevels()
	if err != nil {
		return nil, fmt.Errorf("load embedded levels: %w", err)
	}
	return &Resolver{
		levels: lf,
		games:  map[string]*colorsort.Save{},
	}, nil
}

package colorsort

import (
	"fmt"
	"math/rand"
)

var colorPalette = []string{"red", "blue", "green", "yellow", "purple", "orange", "pink", "cyan", "gray", "brown", "lime", "teal"}

// GenerateConfig configures puzzle generation.
type GenerateConfig struct {
	Difficulty string // "easy", "medium", or "hard"
	Colors     int    // number of distinct colors
	Capacity   int    // tube capacity
	Empty      int    // number of empty tubes
	Seed       int64  // RNG seed
}

// Generate creates a random puzzle with the given configuration.
// The generated puzzle is not guaranteed to be solvable; use Solve() to verify.
func Generate(cfg GenerateConfig) (*Level, error) {
	if cfg.Colors > len(colorPalette) {
		return nil, fmt.Errorf("too many colors: %d > %d", cfg.Colors, len(colorPalette))
	}
	if cfg.Capacity <= 0 {
		return nil, fmt.Errorf("invalid capacity: %d", cfg.Capacity)
	}
	if cfg.Colors <= 0 {
		return nil, fmt.Errorf("invalid colors: %d", cfg.Colors)
	}
	if cfg.Difficulty == "" {
		cfg.Difficulty = "easy"
	}

	rng := rand.New(rand.NewSource(cfg.Seed))

	// Create segments: colors * capacity of each color.
	segs := make([]string, 0, cfg.Colors*cfg.Capacity)
	for c := 0; c < cfg.Colors; c++ {
		for i := 0; i < cfg.Capacity; i++ {
			segs = append(segs, colorPalette[c])
		}
	}

	// Shuffle segments.
	rng.Shuffle(len(segs), func(i, j int) { segs[i], segs[j] = segs[j], segs[i] })

	// Deal into tubes.
	tubes := make([]Tube, 0, cfg.Colors+cfg.Empty)
	for c := 0; c < cfg.Colors; c++ {
		tube := Tube(append([]string{}, segs[c*cfg.Capacity:(c+1)*cfg.Capacity]...))
		tubes = append(tubes, tube)
	}
	for i := 0; i < cfg.Empty; i++ {
		tubes = append(tubes, Tube{})
	}

	return &Level{
		ID:         0, // caller should assign
		Difficulty: cfg.Difficulty,
		Capacity:   cfg.Capacity,
		Tubes:      tubes,
	}, nil
}

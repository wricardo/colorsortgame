package colorsort

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func LoadLevels(path string) (*LevelsFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read levels file: %w", err)
	}
	var lf LevelsFile
	if err := json.Unmarshal(b, &lf); err != nil {
		return nil, fmt.Errorf("parse levels file: %w", err)
	}
	return &lf, nil
}

func LoadSave(path string) (*Save, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read save file: %w", err)
	}
	var s Save
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse save file: %w", err)
	}
	return &s, nil
}

func WriteSave(path string, s *Save) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// ParseMoves parses "1-4,2-3,3-1" into a slice of Move.
func ParseMoves(s string) ([]Move, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("no moves given")
	}
	parts := strings.Split(s, ",")
	moves := make([]Move, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		tuple := strings.Split(p, "-")
		if len(tuple) != 2 {
			return nil, fmt.Errorf("invalid move tuple %q, want from-to", p)
		}
		from, err := strconv.Atoi(strings.TrimSpace(tuple[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid move tuple %q: %w", p, err)
		}
		to, err := strconv.Atoi(strings.TrimSpace(tuple[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid move tuple %q: %w", p, err)
		}
		moves = append(moves, Move{From: from, To: to})
	}
	return moves, nil
}

package colorsort

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed levels.json
var embeddedLevelsJSON []byte

// DefaultLevels returns the bundled 30-level set compiled into the binary,
// so the CLI runs standalone with no external levels.json required.
func DefaultLevels() (*LevelsFile, error) {
	var lf LevelsFile
	if err := json.Unmarshal(embeddedLevelsJSON, &lf); err != nil {
		return nil, fmt.Errorf("parse embedded levels: %w", err)
	}
	return &lf, nil
}

// LoadLevelsOrDefault loads levels from path, or falls back to the embedded
// default set when path is empty.
func LoadLevelsOrDefault(path string) (*LevelsFile, error) {
	if path == "" {
		return DefaultLevels()
	}
	return LoadLevels(path)
}

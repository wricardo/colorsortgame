package colorsort

import "testing"

func TestDefaultLevels(t *testing.T) {
	lf, err := DefaultLevels()
	if err != nil {
		t.Fatalf("DefaultLevels: %v", err)
	}
	if len(lf.Levels) != 30 {
		t.Fatalf("expected 30 embedded levels, got %d", len(lf.Levels))
	}
	for _, l := range lf.Levels {
		if l.Difficulty == "" {
			t.Fatalf("level %d missing difficulty", l.ID)
		}
	}
}

// TestDefaultLevelsAllSolvable is the same guarantee the level generator
// checks at authoring time, re-verified here so a bad edit to levels.json
// fails the test suite instead of silently shipping an unsolvable level.
func TestDefaultLevelsAllSolvable(t *testing.T) {
	lf, err := DefaultLevels()
	if err != nil {
		t.Fatalf("DefaultLevels: %v", err)
	}
	for _, l := range lf.Levels {
		res := Solve(&l)
		if res.Unknown {
			t.Fatalf("level %d: solvability unknown (search cap hit)", l.ID)
		}
		if !res.Solvable {
			t.Fatalf("level %d: not solvable", l.ID)
		}
	}
}

func TestLoadLevelsOrDefault(t *testing.T) {
	lf, err := LoadLevelsOrDefault("")
	if err != nil {
		t.Fatalf("LoadLevelsOrDefault(\"\"): %v", err)
	}
	if len(lf.Levels) != 30 {
		t.Fatalf("expected embedded fallback with 30 levels, got %d", len(lf.Levels))
	}

	if _, err := LoadLevelsOrDefault("/nonexistent/levels.json"); err == nil {
		t.Fatal("expected error loading a nonexistent explicit path")
	}
}

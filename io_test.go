package colorsort

import (
	"path/filepath"
	"testing"
)

func TestParseMoves(t *testing.T) {
	moves, err := ParseMoves("1-4, 2-3,3-1")
	if err != nil {
		t.Fatalf("ParseMoves: %v", err)
	}
	want := []Move{{From: 1, To: 4}, {From: 2, To: 3}, {From: 3, To: 1}}
	if len(moves) != len(want) {
		t.Fatalf("expected %d moves, got %d: %v", len(want), len(moves), moves)
	}
	for i, m := range moves {
		if m != want[i] {
			t.Fatalf("move %d = %v, want %v", i, m, want[i])
		}
	}

	cases := []string{"", "1-2,bad", "1", "1-2-3"}
	for _, c := range cases {
		if _, err := ParseMoves(c); err == nil {
			t.Fatalf("ParseMoves(%q): expected error", c)
		}
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.json")

	s := NewSave(trivialLevel(), "")
	if err := s.ApplyMove(1, 3); err != nil {
		t.Fatal(err)
	}
	if err := WriteSave(path, s); err != nil {
		t.Fatalf("WriteSave: %v", err)
	}

	loaded, err := LoadSave(path)
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if loaded.Moves != s.Moves || loaded.Solved != s.Solved {
		t.Fatalf("round-tripped save mismatch: got %+v, want %+v", loaded, s)
	}
}

func TestLoadLevelsMissingFile(t *testing.T) {
	if _, err := LoadLevels("/nonexistent/levels.json"); err == nil {
		t.Fatal("expected error loading a nonexistent file")
	}
}

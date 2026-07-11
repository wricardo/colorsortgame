package colorsort

import (
	"encoding/json"
	"os"
	"testing"
)

// trivialLevel is one move from solved: pouring tube 1 onto tube 2 (both
// single red segments) fills tube 2 and empties tube 1.
func trivialLevel() *Level {
	return &Level{
		ID:       3,
		Capacity: 2,
		Tubes:    []Tube{{"red"}, {"red"}, {}},
	}
}

func stuckLevel() *Level {
	return &Level{
		ID:       2,
		Capacity: 2,
		Tubes:    []Tube{{"red", "blue"}, {"blue", "red"}},
	}
}

func TestFindLevel(t *testing.T) {
	lf := &LevelsFile{Levels: []Level{{ID: 1}, {ID: 2}}}

	if l, err := FindLevel(lf, 2); err != nil || l.ID != 2 {
		t.Fatalf("FindLevel(2) = %v, %v", l, err)
	}
	if _, err := FindLevel(lf, 99); err == nil {
		t.Fatal("expected error for missing level")
	}
}

func TestApplyMoveWins(t *testing.T) {
	s := NewSave(trivialLevel(), "")
	if s.Solved || s.Stuck {
		t.Fatalf("fresh save should be neither solved nor stuck: %+v", s)
	}

	if err := s.ApplyMove(1, 2); err != nil {
		t.Fatalf("ApplyMove(1,2): %v", err)
	}
	if !s.Solved {
		t.Fatalf("expected solved after winning move: %+v", s)
	}
	if s.Moves != 1 || len(s.History) != 1 {
		t.Fatalf("expected 1 recorded move, got moves=%d history=%v", s.Moves, s.History)
	}
	if len(s.Tubes[0]) != 0 || len(s.Tubes[1]) != 2 {
		t.Fatalf("unexpected tube state after move: %v", s.Tubes)
	}
}

func TestApplyMoveIllegal(t *testing.T) {
	cases := []struct {
		name    string
		from    int
		to      int
		wantErr bool
	}{
		{"self-pour", 1, 1, true},
		{"out of range low", 0, 1, true},
		{"out of range high", 1, 99, true},
		{"empty source", 3, 1, true},
		{"legal", 1, 2, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fresh := NewSave(trivialLevel(), "")
			err := fresh.ApplyMove(c.from, c.to)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestApplyMoveColorMismatch(t *testing.T) {
	s := &Save{Capacity: 2, Tubes: []Tube{{"red"}, {"blue"}}}
	if err := s.ApplyMove(1, 2); err == nil {
		t.Fatal("expected error pouring onto a tube topped with a different color")
	}
}

func TestApplyMoveFullDestination(t *testing.T) {
	s := &Save{Capacity: 2, Tubes: []Tube{{"red"}, {"blue", "blue"}}}
	if err := s.ApplyMove(1, 2); err == nil {
		t.Fatal("expected error pouring onto a full tube")
	}
}

func TestNewSaveDetectsStuck(t *testing.T) {
	s := NewSave(stuckLevel(), "")
	if !s.Stuck {
		t.Fatalf("expected stuck level to be flagged stuck at start: %+v", s)
	}
	if s.Solved {
		t.Fatal("stuck level should not be solved")
	}
}

func TestUndo(t *testing.T) {
	s := NewSave(trivialLevel(), "")
	if err := s.Undo(); err == nil {
		t.Fatal("expected error undoing with empty history")
	}

	// Undo replays from LevelsPath, so give it a real backing file via a
	// temp levels.json rather than relying on the embedded default.
	tmp := t.TempDir() + "/levels.json"
	lf := &LevelsFile{Levels: []Level{*trivialLevel()}}
	b, err := json.Marshal(lf)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		t.Fatal(err)
	}

	level, err := FindLevel(lf, 3)
	if err != nil {
		t.Fatal(err)
	}
	s = NewSave(level, tmp)
	if err := s.ApplyMove(1, 2); err != nil {
		t.Fatal(err)
	}
	if !s.Solved {
		t.Fatal("expected solved before undo")
	}

	if err := s.Undo(); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if s.Solved {
		t.Fatal("expected not solved after undo")
	}
	if s.Moves != 0 || len(s.History) != 0 {
		t.Fatalf("expected empty history after undoing the only move: moves=%d history=%v", s.Moves, s.History)
	}
	if len(s.Tubes[0]) != 1 || len(s.Tubes[1]) != 1 {
		t.Fatalf("unexpected tube state after undo: %v", s.Tubes)
	}
}

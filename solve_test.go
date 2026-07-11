package colorsort

import "testing"

func TestSolveAlreadySolved(t *testing.T) {
	level := &Level{Capacity: 2, Tubes: []Tube{{"red", "red"}, {}}}
	res := Solve(level)
	if !res.Solvable || res.MinMoves != 0 {
		t.Fatalf("expected trivially solved level, got %+v", res)
	}
}

func TestSolveTrivial(t *testing.T) {
	res := Solve(trivialLevel())
	if !res.Solvable {
		t.Fatalf("expected solvable, got %+v", res)
	}
	if res.MinMoves != 1 {
		t.Fatalf("expected 1-move solution, got %d moves: %v", res.MinMoves, res.Path)
	}
	if len(res.Path) != 1 || res.Path[0] != (Move{From: 1, To: 2}) {
		t.Fatalf("unexpected path: %v", res.Path)
	}
}

func TestSolveUnsolvable(t *testing.T) {
	res := Solve(stuckLevel())
	if res.Solvable {
		t.Fatalf("expected not solvable, got %+v", res)
	}
	if res.Unknown {
		t.Fatal("small stuck level should be provably unsolvable, not unknown")
	}
}

// TestSolvePathIsValid replays the path Solve returns and checks it actually
// reaches a solved state, guarding against the BFS returning a bogus path.
func TestSolvePathIsValid(t *testing.T) {
	level := &Level{
		Capacity: 4,
		Tubes: []Tube{
			{"red", "red", "blue", "blue"},
			{"blue", "blue", "red", "red"},
			{},
			{},
		},
	}
	res := Solve(level)
	if !res.Solvable {
		t.Fatalf("expected solvable, got %+v", res)
	}

	s := NewSave(level, "")
	for _, m := range res.Path {
		if err := s.ApplyMove(m.From, m.To); err != nil {
			t.Fatalf("replaying solved path failed at move %+v: %v", m, err)
		}
	}
	if !s.Solved {
		t.Fatalf("replaying the returned path did not solve the level: %+v", s)
	}
	if s.Moves != res.MinMoves {
		t.Fatalf("move count mismatch: replay=%d reported=%d", s.Moves, res.MinMoves)
	}
}

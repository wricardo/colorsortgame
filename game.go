// Package colorsort is a non-interactive water-sort puzzle engine: every
// action loads state from JSON, mutates it, and writes it back. See
// cmd/colorsort for the CLI built on top of this library.
package colorsort

import "fmt"

type Tube []string

type Level struct {
	ID         int    `json:"id"`
	Difficulty string `json:"difficulty"`
	Capacity   int    `json:"tube_capacity"`
	Tubes      []Tube `json:"tubes"`
}

type LevelsFile struct {
	Levels []Level `json:"levels"`
}

type Move struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type Save struct {
	LevelID    int    `json:"level_id"`
	LevelsPath string `json:"levels_path"`
	Capacity   int    `json:"capacity"`
	Tubes      []Tube `json:"tubes"`
	Moves      int    `json:"moves"`
	History    []Move `json:"history"`
	Solved     bool   `json:"solved"`
	Stuck      bool   `json:"stuck"`
}

func FindLevel(lf *LevelsFile, id int) (*Level, error) {
	for i := range lf.Levels {
		if lf.Levels[i].ID == id {
			return &lf.Levels[i], nil
		}
	}
	return nil, fmt.Errorf("level %d not found", id)
}

func NewSave(l *Level, levelsPath string) *Save {
	tubes := make([]Tube, len(l.Tubes))
	for i, t := range l.Tubes {
		tubes[i] = append(Tube{}, t...)
	}
	s := &Save{
		LevelID:    l.ID,
		LevelsPath: levelsPath,
		Capacity:   l.Capacity,
		Tubes:      tubes,
		Moves:      0,
		History:    []Move{},
		Solved:     false,
	}
	s.Stuck = s.isStuck()
	return s
}

// ApplyMove pours from tube `from` onto tube `to` (1-indexed), following
// water-sort rules: move the contiguous same-color run off the top of
// `from` onto `to`, limited by `to`'s remaining capacity.
func (s *Save) ApplyMove(from, to int) error {
	if from < 1 || from > len(s.Tubes) || to < 1 || to > len(s.Tubes) {
		return fmt.Errorf("tube index out of range")
	}
	if from == to {
		return fmt.Errorf("cannot pour tube into itself")
	}
	src := s.Tubes[from-1]
	dst := s.Tubes[to-1]

	if len(src) == 0 {
		return fmt.Errorf("source tube %d is empty", from)
	}
	if len(dst) >= s.Capacity {
		return fmt.Errorf("destination tube %d is full", to)
	}

	topColor := src[len(src)-1]
	if len(dst) > 0 && dst[len(dst)-1] != topColor {
		return fmt.Errorf("color mismatch: top of tube %d is %s, top of tube %d is %s", from, topColor, to, dst[len(dst)-1])
	}

	// count contiguous run of topColor from the top of src
	run := 0
	for i := len(src) - 1; i >= 0 && src[i] == topColor; i-- {
		run++
	}

	space := s.Capacity - len(dst)
	n := run
	if space < n {
		n = space
	}

	s.Tubes[from-1] = src[:len(src)-n]
	s.Tubes[to-1] = append(dst, src[len(src)-n:]...)

	s.Moves++
	s.History = append(s.History, Move{From: from, To: to})
	s.Solved = s.isSolved()
	s.Stuck = !s.Solved && s.isStuck()
	return nil
}

func (s *Save) isSolved() bool {
	for _, t := range s.Tubes {
		if len(t) == 0 {
			continue
		}
		if len(t) != s.Capacity {
			return false
		}
		c := t[0]
		for _, x := range t {
			if x != c {
				return false
			}
		}
	}
	return true
}

// canPour reports whether pouring from `from` onto `to` (1-indexed) is legal,
// without mutating state.
func (s *Save) canPour(from, to int) bool {
	if from == to {
		return false
	}
	src := s.Tubes[from-1]
	dst := s.Tubes[to-1]
	if len(src) == 0 || len(dst) >= s.Capacity {
		return false
	}
	topColor := src[len(src)-1]
	if len(dst) > 0 && dst[len(dst)-1] != topColor {
		return false
	}
	// pouring a tube that's already a single completed color onto an empty
	// tube changes nothing meaningful; still counts as a legal move for
	// stuck-detection purposes, so no special-case needed here.
	return true
}

// isStuck reports whether no legal move remains. Only meaningful when not
// already solved.
func (s *Save) isStuck() bool {
	for from := 1; from <= len(s.Tubes); from++ {
		for to := 1; to <= len(s.Tubes); to++ {
			if s.canPour(from, to) {
				return false
			}
		}
	}
	return true
}

func (s *Save) Undo() error {
	if len(s.History) == 0 {
		return fmt.Errorf("no moves to undo")
	}
	// replay history minus last move onto a fresh copy is simplest & safest.
	last := s.History[:len(s.History)-1]
	lf, err := LoadLevels(s.LevelsPath)
	if err != nil {
		return err
	}
	level, err := FindLevel(lf, s.LevelID)
	if err != nil {
		return err
	}
	fresh := NewSave(level, s.LevelsPath)
	for _, m := range last {
		if err := fresh.ApplyMove(m.From, m.To); err != nil {
			return fmt.Errorf("replay failed: %w", err)
		}
	}
	*s = *fresh
	return nil
}

func PrintBoard(s *Save) {
	fmt.Printf("Level %d | moves: %d | solved: %v\n", s.LevelID, s.Moves, s.Solved)
	for i, t := range s.Tubes {
		fmt.Printf("%2d: %v\n", i+1, t)
	}
	if s.Solved {
		fmt.Println("*** YOU WIN ***")
	} else if s.Stuck {
		fmt.Println("*** STUCK - no legal moves left ***")
	}
}

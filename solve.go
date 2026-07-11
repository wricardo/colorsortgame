package colorsort

import (
	"sort"
	"strings"
)

// SolveStateCap is a pure safety valve against pathological/adversarial
// levels; with canonical-key dedup and parent-pointer path storage the
// search is memory-cheap per state, so real hand-designed levels finish
// long before this is hit.
const SolveStateCap = 20000000

func cloneTubesState(tubes []Tube) []Tube {
	out := make([]Tube, len(tubes))
	for i, t := range tubes {
		out[i] = append(Tube{}, t...)
	}
	return out
}

func tubeStr(t Tube) string {
	var b strings.Builder
	for _, c := range t {
		b.WriteString(c)
		b.WriteByte(',')
	}
	return b.String()
}

// tubesKey canonicalizes the state for visited-set dedup: tubes are
// anonymous (only distinguished by index for move purposes), so two states
// differing only by a permutation of tubes (e.g. which of two empty tubes
// is "tube 3" vs "tube 4") are the same state and must collapse to one key.
func tubesKey(tubes []Tube) string {
	strs := make([]string, len(tubes))
	for i, t := range tubes {
		strs[i] = tubeStr(t)
	}
	sort.Strings(strs)
	return strings.Join(strs, "|")
}

func isSolvedTubes(tubes []Tube, capacity int) bool {
	for _, t := range tubes {
		if len(t) == 0 {
			continue
		}
		if len(t) != capacity {
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

func canPourTubes(tubes []Tube, capacity, from, to int) bool {
	if from == to {
		return false
	}
	src := tubes[from]
	dst := tubes[to]
	if len(src) == 0 || len(dst) >= capacity {
		return false
	}
	top := src[len(src)-1]
	if len(dst) > 0 && dst[len(dst)-1] != top {
		return false
	}
	// pouring a lone-color tube onto an empty tube is a no-op move
	// (same shape, different index); still legal, harmless for search.
	return true
}

func pourTubes(tubes []Tube, capacity, from, to int) []Tube {
	next := cloneTubesState(tubes)
	src := next[from]
	dst := next[to]
	top := src[len(src)-1]
	run := 0
	for i := len(src) - 1; i >= 0 && src[i] == top; i-- {
		run++
	}
	space := capacity - len(dst)
	n := run
	if space < n {
		n = space
	}
	next[from] = src[:len(src)-n]
	next[to] = append(dst, src[len(src)-n:]...)
	return next
}

// SolveResult describes the outcome of a solvability search.
type SolveResult struct {
	Solvable bool
	Unknown  bool // state cap reached before exhausting search space
	MinMoves int
	Path     []Move // 0-indexed tube pairs, one per move, shortest found
}

// parentLink records how a state was first reached, for path reconstruction
// without storing a full move-list on every queued node.
type parentLink struct {
	key  string
	move Move
}

// Solve runs a breadth-first search over the level's state space to find
// the shortest sequence of moves to a solved state, if any exists. States
// are deduped via a canonical key (tubesKey) and paths are reconstructed
// via parent pointers rather than carrying a move-list per queued node, so
// memory per visited state is O(1) instead of O(depth).
func Solve(l *Level) SolveResult {
	start := cloneTubesState(l.Tubes)
	if isSolvedTubes(start, l.Capacity) {
		return SolveResult{Solvable: true, MinMoves: 0}
	}

	startKey := tubesKey(start)
	parent := map[string]parentLink{}
	visited := map[string]bool{startKey: true}
	queue := []([]Tube){start}

	visitedCount := 1
	n := len(l.Tubes)

	buildPath := func(key string) []Move {
		var rev []Move
		for key != startKey {
			link := parent[key]
			rev = append(rev, link.move)
			key = link.key
		}
		path := make([]Move, len(rev))
		for i, m := range rev {
			path[len(rev)-1-i] = m
		}
		return path
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curKey := tubesKey(cur)

		for from := 0; from < n; from++ {
			for to := 0; to < n; to++ {
				if !canPourTubes(cur, l.Capacity, from, to) {
					continue
				}
				next := pourTubes(cur, l.Capacity, from, to)
				key := tubesKey(next)
				if visited[key] {
					continue
				}
				visited[key] = true
				visitedCount++
				parent[key] = parentLink{key: curKey, move: Move{From: from + 1, To: to + 1}}

				if isSolvedTubes(next, l.Capacity) {
					path := buildPath(key)
					return SolveResult{Solvable: true, MinMoves: len(path), Path: path}
				}

				if visitedCount > SolveStateCap {
					return SolveResult{Unknown: true}
				}

				queue = append(queue, next)
			}
		}
	}

	return SolveResult{Solvable: false}
}

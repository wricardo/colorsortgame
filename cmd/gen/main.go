package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
)

type Tube []string

var palette = []string{"red", "blue", "green", "yellow", "purple", "orange", "pink", "cyan", "gray", "brown", "lime", "teal"}

func main() {
	id := flag.Int("id", 1, "level id")
	difficulty := flag.String("difficulty", "easy", "difficulty label: easy, medium, or hard")
	colors := flag.Int("colors", 3, "number of colors")
	capacity := flag.Int("capacity", 4, "tube capacity")
	empty := flag.Int("empty", 2, "number of empty tubes")
	seed := flag.Int64("seed", 1, "rng seed")
	flag.Parse()

	if *colors > len(palette) {
		fmt.Fprintln(os.Stderr, "too many colors for palette")
		os.Exit(1)
	}

	rng := rand.New(rand.NewSource(*seed))

	// standard water-sort generation: colors*capacity segments, shuffled,
	// dealt into exactly `colors` tubes (filled to capacity), plus `empty`
	// genuinely empty spare tubes.
	segs := make([]string, 0, *colors**capacity)
	for c := 0; c < *colors; c++ {
		for i := 0; i < *capacity; i++ {
			segs = append(segs, palette[c])
		}
	}
	rng.Shuffle(len(segs), func(i, j int) { segs[i], segs[j] = segs[j], segs[i] })

	tubes := make([]Tube, 0, *colors+*empty)
	for c := 0; c < *colors; c++ {
		tubes = append(tubes, append(Tube{}, segs[c**capacity:(c+1)**capacity]...))
	}
	for i := 0; i < *empty; i++ {
		tubes = append(tubes, Tube{})
	}

	out := struct {
		ID         int    `json:"id"`
		Difficulty string `json:"difficulty"`
		Capacity   int    `json:"tube_capacity"`
		Tubes      []Tube `json:"tubes"`
	}{ID: *id, Difficulty: *difficulty, Capacity: *capacity, Tubes: tubes}

	b, _ := json.Marshal(out)
	fmt.Println(string(b))
}

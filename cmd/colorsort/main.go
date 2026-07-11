package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wricardo/colorsortgame"
)

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: colorsort <new|move|move-bulk|undo|show|status|list|reset|solvable> [flags]")
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "new", "reset":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		levelID := fs.Int("level", 0, "level id to load")
		levelsPath := fs.String("levels", "./levels.json", "path to levels.json")
		savePath := fs.String("save", "./save.json", "path to save.json")
		_ = fs.Parse(args)

		lf, err := colorsort.LoadLevels(*levelsPath)
		if err != nil {
			fail(err)
		}
		level, err := colorsort.FindLevel(lf, *levelID)
		if err != nil {
			fail(err)
		}

		res := colorsort.Solve(level)
		if res.Unknown {
			fmt.Fprintf(os.Stderr, "warning: solvability unknown (search cap of %d states reached), starting anyway\n", colorsort.SolveStateCap)
		} else if !res.Solvable {
			fail(fmt.Errorf("level %d is not solvable, refusing to start", *levelID))
		}

		s := colorsort.NewSave(level, *levelsPath)
		if err := colorsort.WriteSave(*savePath, s); err != nil {
			fail(err)
		}
		colorsort.PrintBoard(s)

	case "move":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		from := fs.Int("from", 0, "source tube (1-indexed)")
		to := fs.Int("to", 0, "destination tube (1-indexed)")
		savePath := fs.String("save", "./save.json", "path to save.json")
		_ = fs.Parse(args)

		s, err := colorsort.LoadSave(*savePath)
		if err != nil {
			fail(err)
		}
		if err := s.ApplyMove(*from, *to); err != nil {
			fail(err)
		}
		if err := colorsort.WriteSave(*savePath, s); err != nil {
			fail(err)
		}
		colorsort.PrintBoard(s)
		if s.Stuck {
			os.Exit(2)
		}

	case "move-bulk":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		moves := fs.String("moves", "", "comma-separated from-to tuples, e.g. 1-4,2-3,3-1")
		savePath := fs.String("save", "./save.json", "path to save.json")
		_ = fs.Parse(args)

		tuples, err := colorsort.ParseMoves(*moves)
		if err != nil {
			fail(err)
		}

		s, err := colorsort.LoadSave(*savePath)
		if err != nil {
			fail(err)
		}

		var applyErr error
		var failedAt int
		for i, m := range tuples {
			if err := s.ApplyMove(m.From, m.To); err != nil {
				applyErr = err
				failedAt = i + 1
				break
			}
			if s.Solved || s.Stuck {
				break
			}
		}

		if err := colorsort.WriteSave(*savePath, s); err != nil {
			fail(err)
		}
		colorsort.PrintBoard(s)
		if applyErr != nil {
			fail(fmt.Errorf("move %d (%d-%d): %w", failedAt, tuples[failedAt-1].From, tuples[failedAt-1].To, applyErr))
		}
		if s.Stuck {
			os.Exit(2)
		}

	case "undo":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		savePath := fs.String("save", "./save.json", "path to save.json")
		_ = fs.Parse(args)

		s, err := colorsort.LoadSave(*savePath)
		if err != nil {
			fail(err)
		}
		if err := s.Undo(); err != nil {
			fail(err)
		}
		if err := colorsort.WriteSave(*savePath, s); err != nil {
			fail(err)
		}
		colorsort.PrintBoard(s)

	case "show", "status":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		savePath := fs.String("save", "./save.json", "path to save.json")
		_ = fs.Parse(args)

		s, err := colorsort.LoadSave(*savePath)
		if err != nil {
			fail(err)
		}
		colorsort.PrintBoard(s)

	case "solvable":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		levelID := fs.Int("level", 0, "level id to check")
		levelsPath := fs.String("levels", "./levels.json", "path to levels.json")
		showPath := fs.Bool("path", false, "print the shortest solving move sequence")
		_ = fs.Parse(args)

		lf, err := colorsort.LoadLevels(*levelsPath)
		if err != nil {
			fail(err)
		}
		level, err := colorsort.FindLevel(lf, *levelID)
		if err != nil {
			fail(err)
		}

		res := colorsort.Solve(level)
		switch {
		case res.Unknown:
			fmt.Printf("level %d: unknown (search cap of %d states reached)\n", *levelID, colorsort.SolveStateCap)
			os.Exit(2)
		case res.Solvable:
			fmt.Printf("level %d: solvable in %d moves\n", *levelID, res.MinMoves)
			if *showPath {
				parts := make([]string, len(res.Path))
				for i, m := range res.Path {
					parts[i] = fmt.Sprintf("%d-%d", m.From, m.To)
				}
				fmt.Println(strings.Join(parts, ","))
			}
		default:
			fmt.Printf("level %d: not solvable\n", *levelID)
			os.Exit(1)
		}

	case "list":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		levelsPath := fs.String("levels", "./levels.json", "path to levels.json")
		_ = fs.Parse(args)

		lf, err := colorsort.LoadLevels(*levelsPath)
		if err != nil {
			fail(err)
		}
		for _, l := range lf.Levels {
			fmt.Printf("level %d [%s]: %d tubes, capacity %d\n", l.ID, l.Difficulty, len(l.Tubes), l.Capacity)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		os.Exit(1)
	}
}

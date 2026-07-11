package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/wricardo/colorsortgame"
)

// jsonOut is set from each subcommand's --json flag right after parsing, so
// fail() can format errors consistently with the command's chosen output mode.
var jsonOut bool

func fail(err error) {
	if jsonOut {
		b, _ := json.Marshal(struct {
			Error string `json:"error"`
		}{Error: err.Error()})
		fmt.Fprintln(os.Stderr, string(b))
	} else {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	os.Exit(1)
}

// output prints v as indented JSON when asJSON is set, otherwise calls text.
func output(asJSON bool, v any, text func()) {
	if asJSON {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			fail(err)
		}
		fmt.Println(string(b))
		return
	}
	text()
}

type solvableResult struct {
	Level    int      `json:"level"`
	Solvable bool     `json:"solvable"`
	Unknown  bool     `json:"unknown"`
	MinMoves int      `json:"min_moves,omitempty"`
	Path     []string `json:"path,omitempty"`
}

const usage = `colorsort - non-interactive water-sort puzzle CLI

Every command is a single, independent invocation: it loads ./save.json (or
--save PATH), applies one action, writes the result back, and exits. There
is no REPL and no long-running process.

Usage:
  colorsort <command> [flags]

Commands:
  list
        List every level in levels.json: id, difficulty, tube count, capacity.
        Flags: --levels PATH (default ./levels.json), --json

  solvable --level N [--path]
        Run an exhaustive search to determine whether level N is solvable,
        without touching any save file. Exit code 0 if solvable, 1 if
        provably not solvable, 2 if the search-state cap was hit before a
        definitive answer (see SolveStateCap).
        Flags: --level N (required), --levels PATH, --path (also print a
        shortest solving move sequence), --json

  new --level N
        Start level N: runs the same solvability search as 'solvable' first
        and refuses to start (exit 1) if the level is provably unsolvable.
        Writes a fresh save file, overwriting any existing one.
        Flags: --level N (required), --levels PATH, --save PATH, --json

  reset --level N
        Alias for 'new' — restart level N from scratch.
        Flags: same as 'new'.

  move --from A --to B
        Pour tube A onto tube B (1-indexed). Moves the contiguous run of
        same-colored segments off the top of A onto B, limited by B's
        remaining capacity; legal only if B is empty or its top segment
        matches A's top color. Exit code 2 if this move leaves the board
        stuck (no legal moves remain and it isn't solved).
        Flags: --from N (required), --to N (required), --save PATH, --json

  move-bulk --moves "1-4,2-3,3-1"
        Apply a comma-separated sequence of from-to move tuples in order.
        Stops at the first illegal move, still persists progress made up to
        that point, and reports which move (1-indexed within the batch)
        failed. Also stops early if the board becomes solved or stuck.
        Exit code 2 if stuck.
        Flags: --moves "A-B,C-D,..." (required), --save PATH, --json

  undo
        Undo the last move by replaying history from the level's start.
        Flags: --save PATH, --json

  show / status
        Print the current board without changing anything.
        Flags: --save PATH, --json

Global notes:
  --json on any command prints structured JSON on stdout instead of the
  text board (the full Save object for game commands, a
  {level,solvable,unknown,min_moves,path} object for 'solvable', or the
  level array for 'list'). Errors are emitted as {"error": "..."} on
  stderr too when --json is set. Exit codes are unchanged either way.

  Run 'colorsort <command> -h' for that command's flag list.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return

	case "new", "reset":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		levelID := fs.Int("level", 0, "level id to load")
		levelsPath := fs.String("levels", "./levels.json", "path to levels.json")
		savePath := fs.String("save", "./save.json", "path to save.json")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(args)
		jsonOut = *asJSON

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
		output(*asJSON, s, func() { colorsort.PrintBoard(s) })

	case "move":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		from := fs.Int("from", 0, "source tube (1-indexed)")
		to := fs.Int("to", 0, "destination tube (1-indexed)")
		savePath := fs.String("save", "./save.json", "path to save.json")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(args)
		jsonOut = *asJSON

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
		output(*asJSON, s, func() { colorsort.PrintBoard(s) })
		if s.Stuck {
			os.Exit(2)
		}

	case "move-bulk":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		moves := fs.String("moves", "", "comma-separated from-to tuples, e.g. 1-4,2-3,3-1")
		savePath := fs.String("save", "./save.json", "path to save.json")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(args)
		jsonOut = *asJSON

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
		output(*asJSON, s, func() { colorsort.PrintBoard(s) })
		if applyErr != nil {
			fail(fmt.Errorf("move %d (%d-%d): %w", failedAt, tuples[failedAt-1].From, tuples[failedAt-1].To, applyErr))
		}
		if s.Stuck {
			os.Exit(2)
		}

	case "undo":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		savePath := fs.String("save", "./save.json", "path to save.json")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(args)
		jsonOut = *asJSON

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
		output(*asJSON, s, func() { colorsort.PrintBoard(s) })

	case "show", "status":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		savePath := fs.String("save", "./save.json", "path to save.json")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(args)
		jsonOut = *asJSON

		s, err := colorsort.LoadSave(*savePath)
		if err != nil {
			fail(err)
		}
		output(*asJSON, s, func() { colorsort.PrintBoard(s) })

	case "solvable":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		levelID := fs.Int("level", 0, "level id to check")
		levelsPath := fs.String("levels", "./levels.json", "path to levels.json")
		showPath := fs.Bool("path", false, "print the shortest solving move sequence")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(args)
		jsonOut = *asJSON

		lf, err := colorsort.LoadLevels(*levelsPath)
		if err != nil {
			fail(err)
		}
		level, err := colorsort.FindLevel(lf, *levelID)
		if err != nil {
			fail(err)
		}

		res := colorsort.Solve(level)
		result := solvableResult{Level: *levelID, Solvable: res.Solvable, Unknown: res.Unknown, MinMoves: res.MinMoves}
		if res.Solvable && *showPath {
			result.Path = make([]string, len(res.Path))
			for i, m := range res.Path {
				result.Path[i] = fmt.Sprintf("%d-%d", m.From, m.To)
			}
		}

		output(*asJSON, result, func() {
			switch {
			case res.Unknown:
				fmt.Printf("level %d: unknown (search cap of %d states reached)\n", *levelID, colorsort.SolveStateCap)
			case res.Solvable:
				fmt.Printf("level %d: solvable in %d moves\n", *levelID, res.MinMoves)
				if *showPath {
					fmt.Println(strings.Join(result.Path, ","))
				}
			default:
				fmt.Printf("level %d: not solvable\n", *levelID)
			}
		})

		switch {
		case res.Unknown:
			os.Exit(2)
		case !res.Solvable:
			os.Exit(1)
		}

	case "list":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		levelsPath := fs.String("levels", "./levels.json", "path to levels.json")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(args)
		jsonOut = *asJSON

		lf, err := colorsort.LoadLevels(*levelsPath)
		if err != nil {
			fail(err)
		}
		output(*asJSON, lf.Levels, func() {
			for _, l := range lf.Levels {
				fmt.Printf("level %d [%s]: %d tubes, capacity %d\n", l.ID, l.Difficulty, len(l.Tubes), l.Capacity)
			}
		})

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q, run 'colorsort help' for usage\n", cmd)
		os.Exit(1)
	}
}

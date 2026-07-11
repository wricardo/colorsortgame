package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/wricardo/colorsortgame"
	"github.com/wricardo/colorsortgame/graphqlapi"
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
	Level    int  `json:"level"`
	Solvable bool `json:"solvable"`
	Unknown  bool `json:"unknown"`
}

const usage = `colorsort - non-interactive water-sort puzzle CLI

Every command is a single, independent invocation backed by a GraphQL API.
Commands either connect to a remote GraphQL server (--api URL) or spawn an
ephemeral local server (default). There is no REPL, no long-running process,
and no save.json file—games are identified by UUID and held in memory.

Usage:
  colorsort <command> [flags]

Global flags:
  --api URL    point to a running GraphQL server (default: spawn ephemeral local server)

Commands:
  list
        List every level: id, difficulty, tube count, capacity.
        Flags: --json

  solvable --level N
        Query the API: is level N solvable? Exit code 0 if solvable, 1 if
        provably not, 2 if the search-state cap was hit.
        Flags: --level N (required), --json

  new --level N
        Create a new game on level N. API checks solvability first and refuses
        to start if provably unsolvable.
        Flags: --level N (required), --json

  reset --game-id UUID
        Reset a game to level start, clearing all moves and history.
        Flags: --game-id UUID (required), --json

  move --game-id UUID --from A --to B
        Pour tube A onto tube B (1-indexed). Exit code 2 if stuck after move.
        Flags: --game-id UUID (required), --from N (required), --to N (required), --json

  move-bulk --game-id UUID --moves "1-4,2-3,3-1"
        Apply comma-separated move sequence. Stops at first illegal move or
        solved/stuck state. Exit code 2 if stuck.
        Flags: --game-id UUID (required), --moves "..." (required), --json

  undo --game-id UUID
        Undo the last move in a game.
        Flags: --game-id UUID (required), --json

  show / status --game-id UUID
        Print the current board.
        Flags: --game-id UUID (required), --json

  serve
        Start a standalone GraphQL API and web UI server. Blocks until stopped.
        Flags: --port N (default 8080)

Global notes:
  Games are stateful and held in memory; each game has a UUID returned by new.
  --json prints structured JSON on stdout. Errors are emitted as {"error": "..."}
  on stderr when --json is set. Exit codes: 0=success, 1=error, 2=stuck.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	// Parse --api flag globally from the beginning.
	apiURL := ""
	args := os.Args[1:]
	if len(args) >= 2 && args[0] == "--api" {
		apiURL = args[1]
		args = args[2:]
	}

	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	cmd := args[0]
	argsAfterCmd := args[1:]

	// Commands that don't need a client: help, serve.
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		fmt.Print(usage)
		return
	}

	if cmd == "serve" {
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		port := fs.String("port", "8080", "port to listen on")
		_ = fs.Parse(argsAfterCmd)
		if err := graphqlapi.Serve(*port); err != nil {
			fail(err)
		}
		return
	}

	// All other commands need a client.
	cli, err := New(apiURL)
	if err != nil {
		fail(err)
	}
	defer cli.Close()

	switch cmd {

	case "list":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(argsAfterCmd)
		jsonOut = *asJSON

		ctx := context.Background()
		resp, err := cli.Execute(ctx, `query { levels { id difficulty tubeCapacity tubes } }`, nil)
		if err != nil {
			fail(err)
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid response"))
		}
		levelsIface, ok := data["levels"].([]interface{})
		if !ok {
			fail(fmt.Errorf("invalid levels response"))
		}

		var levels []GraphQLLevel
		b, _ := json.Marshal(levelsIface)
		if err := json.Unmarshal(b, &levels); err != nil {
			fail(err)
		}

		output(*asJSON, levels, func() {
			for _, l := range levels {
				fmt.Printf("level %d [%s]: %d tubes, capacity %d\n", l.ID, l.Difficulty, len(l.Tubes), l.TubeCapacity)
			}
		})

	case "new":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		levelID := fs.Int("level", 0, "level id to load")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(argsAfterCmd)
		jsonOut = *asJSON

		if *levelID <= 0 {
			fail(fmt.Errorf("--level required"))
		}

		ctx := context.Background()
		query := `mutation newGame($levelId: Int!) {
			newGame(levelId: $levelId) {
				id levelId difficulty capacity tubes moves solved stuck
				history { from to }
			}
		}`
		vars := map[string]interface{}{"levelId": *levelID}
		resp, err := cli.Execute(ctx, query, vars)
		if err != nil {
			fail(err)
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid response"))
		}
		gameIface, ok := data["newGame"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid newGame response"))
		}

		var game GraphQLGame
		b, _ := json.Marshal(gameIface)
		if err := json.Unmarshal(b, &game); err != nil {
			fail(err)
		}

		save := game.ToSave()
		output(*asJSON, gameIface, func() { colorsort.PrintBoard(save) })

	case "reset":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		gameID := fs.String("game-id", "", "game UUID")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(argsAfterCmd)
		jsonOut = *asJSON

		if *gameID == "" {
			fail(fmt.Errorf("--game-id required"))
		}

		ctx := context.Background()
		query := `mutation resetGame($id: ID!) {
			resetGame(gameId: $id) {
				id levelId difficulty capacity tubes moves solved stuck
				history { from to }
			}
		}`
		vars := map[string]interface{}{"id": *gameID}
		resp, err := cli.Execute(ctx, query, vars)
		if err != nil {
			fail(err)
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid response"))
		}
		gameIface, ok := data["resetGame"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid resetGame response"))
		}

		var game GraphQLGame
		b, _ := json.Marshal(gameIface)
		if err := json.Unmarshal(b, &game); err != nil {
			fail(err)
		}

		save := game.ToSave()
		output(*asJSON, gameIface, func() { colorsort.PrintBoard(save) })

	case "move":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		gameID := fs.String("game-id", "", "game UUID")
		from := fs.Int("from", 0, "source tube (1-indexed)")
		to := fs.Int("to", 0, "destination tube (1-indexed)")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(argsAfterCmd)
		jsonOut = *asJSON

		if *gameID == "" || *from <= 0 || *to <= 0 {
			fail(fmt.Errorf("--game-id, --from, --to all required"))
		}

		ctx := context.Background()
		query := `mutation move($id: ID!, $from: Int!, $to: Int!) {
			move(gameId: $id, from: $from, to: $to) {
				id levelId difficulty capacity tubes moves solved stuck
				history { from to }
			}
		}`
		vars := map[string]interface{}{"id": *gameID, "from": *from, "to": *to}
		resp, err := cli.Execute(ctx, query, vars)
		if err != nil {
			fail(err)
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid response"))
		}
		gameIface, ok := data["move"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid move response"))
		}

		var game GraphQLGame
		b, _ := json.Marshal(gameIface)
		if err := json.Unmarshal(b, &game); err != nil {
			fail(err)
		}

		save := game.ToSave()
		output(*asJSON, gameIface, func() { colorsort.PrintBoard(save) })
		if save.Stuck {
			os.Exit(2)
		}

	case "move-bulk":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		gameID := fs.String("game-id", "", "game UUID")
		moves := fs.String("moves", "", "comma-separated from-to tuples, e.g. 1-4,2-3,3-1")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(argsAfterCmd)
		jsonOut = *asJSON

		if *gameID == "" || *moves == "" {
			fail(fmt.Errorf("--game-id and --moves required"))
		}

		// Validate moves format early
		_, err := colorsort.ParseMoves(*moves)
		if err != nil {
			fail(err)
		}

		ctx := context.Background()
		query := `mutation moveBulk($id: ID!, $moves: String!) {
			moveBulk(gameId: $id, moves: $moves) {
				id levelId difficulty capacity tubes moves solved stuck
				history { from to }
			}
		}`

		vars := map[string]interface{}{"id": *gameID, "moves": *moves}
		resp, err := cli.Execute(ctx, query, vars)
		if err != nil {
			fail(err)
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid response"))
		}
		gameIface, ok := data["moveBulk"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid moveBulk response"))
		}

		var game GraphQLGame
		b, _ := json.Marshal(gameIface)
		if err := json.Unmarshal(b, &game); err != nil {
			fail(err)
		}

		save := game.ToSave()
		output(*asJSON, gameIface, func() { colorsort.PrintBoard(save) })
		if save.Stuck {
			os.Exit(2)
		}

	case "undo":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		gameID := fs.String("game-id", "", "game UUID")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(argsAfterCmd)
		jsonOut = *asJSON

		if *gameID == "" {
			fail(fmt.Errorf("--game-id required"))
		}

		ctx := context.Background()
		query := `mutation undo($id: ID!) {
			undo(gameId: $id) {
				id levelId difficulty capacity tubes moves solved stuck
				history { from to }
			}
		}`
		vars := map[string]interface{}{"id": *gameID}
		resp, err := cli.Execute(ctx, query, vars)
		if err != nil {
			fail(err)
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid response"))
		}
		gameIface, ok := data["undo"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid undo response"))
		}

		var game GraphQLGame
		b, _ := json.Marshal(gameIface)
		if err := json.Unmarshal(b, &game); err != nil {
			fail(err)
		}

		save := game.ToSave()
		output(*asJSON, gameIface, func() { colorsort.PrintBoard(save) })

	case "show", "status":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		gameID := fs.String("game-id", "", "game UUID")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(argsAfterCmd)
		jsonOut = *asJSON

		if *gameID == "" {
			fail(fmt.Errorf("--game-id required"))
		}

		ctx := context.Background()
		query := `query getGame($id: ID!) {
			game(id: $id) {
				id levelId difficulty capacity tubes moves solved stuck
				history { from to }
			}
		}`
		vars := map[string]interface{}{"id": *gameID}
		resp, err := cli.Execute(ctx, query, vars)
		if err != nil {
			fail(err)
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid response"))
		}
		gameIface, ok := data["game"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid game response"))
		}

		var game GraphQLGame
		b, _ := json.Marshal(gameIface)
		if err := json.Unmarshal(b, &game); err != nil {
			fail(err)
		}

		save := game.ToSave()
		output(*asJSON, save, func() { colorsort.PrintBoard(save) })

	case "solvable":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		levelID := fs.Int("level", 0, "level id to check")
		asJSON := fs.Bool("json", false, "output JSON instead of text")
		_ = fs.Parse(argsAfterCmd)
		jsonOut = *asJSON

		if *levelID <= 0 {
			fail(fmt.Errorf("--level required"))
		}

		ctx := context.Background()
		query := `query solvable($levelId: Int!) {
			solvable(levelId: $levelId) {
				solvable unknown
			}
		}`
		vars := map[string]interface{}{"levelId": *levelID}
		resp, err := cli.Execute(ctx, query, vars)
		if err != nil {
			fail(err)
		}

		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid response"))
		}
		solveIface, ok := data["solvable"].(map[string]interface{})
		if !ok {
			fail(fmt.Errorf("invalid solvable response"))
		}

		var solve GraphQLSolveResult
		b, _ := json.Marshal(solveIface)
		if err := json.Unmarshal(b, &solve); err != nil {
			fail(err)
		}

		result := solvableResult{
			Level:    *levelID,
			Solvable: solve.Solvable,
			Unknown:  solve.Unknown,
		}

		output(*asJSON, result, func() {
			switch {
			case solve.Unknown:
				fmt.Printf("level %d: unknown\n", *levelID)
			case solve.Solvable:
				fmt.Printf("level %d: solvable\n", *levelID)
			default:
				fmt.Printf("level %d: not solvable\n", *levelID)
			}
		})

		switch {
		case solve.Unknown:
			os.Exit(2)
		case !solve.Solvable:
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q, run 'colorsort help' for usage\n", cmd)
		os.Exit(1)
	}
}

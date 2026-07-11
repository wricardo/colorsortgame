# colorsortgame

A non-interactive CLI implementation of the water-sort puzzle: pour colored
liquid between tubes until every tube holds a single color (or is empty).

Every command is a single, independent invocation — no REPL, no long-running
process. Game state lives entirely in a JSON save file, read and rewritten
on each call, so the game can be scripted, tested, or driven by any tool
that can shell out and read JSON.

The bundled 30-level set is compiled into the binary (`go:embed`), so a
single binary runs anywhere with no extra files — pass `--levels PATH` to
use a different level set instead.

## Install

```sh
go install github.com/wricardo/colorsortgame/cmd/colorsort@latest
```

Or build from a clone:

```sh
git clone https://github.com/wricardo/colorsortgame.git
cd colorsortgame
go build -o colorsort ./cmd/colorsort
```

## Quick start

```sh
./colorsort list                      # see available levels
./colorsort new --level 1             # start a new game, writes ./save.json
./colorsort move --from 1 --to 4      # pour tube 1 onto tube 4
./colorsort show                      # reprint the current board
./colorsort undo                      # undo the last move
```

## Example: playing level 1

```console
$ ./colorsort new --level 1
Level 1 | moves: 0 | solved: false
 1: [red red green green]
 2: [blue red blue green]
 3: [red blue green blue]
 4: []
 5: []

$ ./colorsort move --from 1 --to 4
Level 1 | moves: 1 | solved: false
 1: [red red]
 2: [blue red blue green]
 3: [red blue green blue]
 4: [green green]
 5: []

$ ./colorsort move-bulk --moves "2-4,3-2,3-4,2-3,2-1,3-2,1-3"
Level 1 | moves: 8 | solved: true
 1: []
 2: [blue blue blue blue]
 3: [red red red red]
 4: [green green green green]
 5: []
*** YOU WIN ***
```

Each command reads `./save.json`, applies one action, writes it back, and
exits — `move-bulk` here just applies the remaining 7 moves of the level's
known 8-move solution (see `colorsort solvable --level 1 --path`) in one call.

## Commands

| Command | Description |
|---|---|
| `list` | List all levels with id, difficulty, tube count, capacity |
| `solvable --level N [--path]` | Check whether level `N` is solvable (exhaustive search); `--path` prints a shortest solution |
| `new --level N` | Start level `N`, refusing to start if it's provably unsolvable |
| `reset --level N` | Same as `new`, restart from scratch |
| `move --from A --to B` | Pour tube `A` onto tube `B` (1-indexed) |
| `move-bulk --moves "1-4,2-3,3-1"` | Apply a comma-separated sequence of moves in order |
| `undo` | Undo the last move |
| `show` / `status` | Print the current board without changing anything |
| `serve --port N` | Start an HTTP server with a GraphQL API and a web UI (default port 8080) |

All commands accept `--save PATH` (default `./save.json`); `list`, `solvable`,
`new`, and `reset` also accept `--levels PATH` (default: the embedded
30-level set; pass a path to load a different `levels.json` instead).

### `--json`

Every command accepts `--json` to print structured JSON on stdout instead of
the human-readable board/text — useful for scripting. `new`/`move`/`move-bulk`/
`undo`/`show` print the full `Save` object; `solvable` prints
`{level, solvable, unknown, min_moves, path}`; `list` prints the level array.
Errors are also emitted as JSON (`{"error": "..."}`) on stderr when `--json`
is set. Exit codes are unchanged either way.

```console
$ ./colorsort move --from 1 --to 4 --json
{
  "level_id": 1,
  "levels_path": "./levels.json",
  "capacity": 4,
  "tubes": [["red", "red"], ["blue", "red", "blue", "green"], ["red", "blue", "green", "blue"], ["green", "green"], []],
  "moves": 1,
  "history": [{"from": 1, "to": 4}],
  "solved": false,
  "stuck": false
}
```

### Exit codes

- `0` — success (or a solved win)
- `1` — illegal move, missing level, or other error
- `2` — the board is stuck (`move`/`move-bulk`), or `solvable` couldn't reach
  a definitive answer within its search-state cap

## Rules

A tube has a fixed `tube_capacity`. Pouring from tube A onto tube B moves
the contiguous run of same-colored segments off the top of A onto B, as
much as fits in B's remaining space, provided B is empty or its top segment
is the same color. A level is solved when every tube is either empty or
full with a single color.

## Levels

`levels.json` (repo root) bundles all levels and is compiled into the binary
via `go:embed` (see `levels.go`); `new`/`reset`/`solvable`/`list` load a
level by `--level` id from the embedded set unless `--levels PATH` points at
a different file. The bundled set has 30 levels split into three difficulty
tiers:

- **easy** (1-10) — 3 colors, 5 tubes, solvable in 5-10 moves
- **medium** (11-20) — 5 colors, 7 tubes, solvable in 11-17 moves
- **hard** (21-30) — 8 colors, 10 tubes, solvable in 22-27 moves

Every bundled level is verified solvable via exhaustive search before being
committed (see `cmd/gen`).

### Level format

```json
{
  "levels": [
    {
      "id": 1,
      "difficulty": "easy",
      "tube_capacity": 4,
      "tubes": [["red", "red", "blue", "blue"], ["blue", "blue", "red", "red"], []]
    }
  ]
}
```

Tubes are listed bottom-to-top. `new`/`reset` run the solvability search
before starting a level and refuse to start (exit code `1`) if it's proven
unsolvable.

## Level generator

`cmd/gen` produces new solvable levels: it deals a shuffled multiset of
color segments into filled tubes plus spare empties. Generated output isn't
guaranteed solvable on its own — verify each candidate with
`colorsort solvable` before adding it to `levels.json`.

```sh
go run ./cmd/gen -id 42 -difficulty medium -colors 5 -capacity 4 -empty 2 -seed 7
```

## GraphQL API and web UI

```sh
./colorsort serve --port 8080
```

Serves three routes:

- `/` — a static web UI (embedded via `go:embed`): pick a level, click a tube
  to select it as the pour source, click another to pour into it
- `/query` — the GraphQL API
- `/playground` — a GraphQL playground for exploring the schema and running
  queries by hand

Games are stateful, held in memory on the server and keyed by an id returned
from `newGame` — there's no equivalent of `--save PATH` here since a single
server can host many concurrent games. Queries: `levels`, `level`,
`solvable`, `game`. Mutations: `newGame`, `move`, `moveBulk`, `undo`,
`resetGame`. Only the embedded level set is available over GraphQL (no
`--levels` equivalent).

The implementation lives in `graphqlapi/` (gqlgen, schema-first): schema in
`graphqlapi/graph/schema.graphqls`, resolvers in
`graphqlapi/graph/schema.resolvers.go`, UI assets in `graphqlapi/static/`.

### Running via mise

```sh
mise run serve   # ./colorsort serve
mise run build   # build the CLI binary
mise run test    # go vet + golangci-lint
```

## Tests

```sh
go test ./...
```

Covers move legality, win/stuck detection, undo, the BFS solver (including
replaying its returned path to confirm it actually solves), and re-verifies
that every bundled level is solvable — so a bad edit to `levels.json` fails
the test suite instead of shipping silently.

## Architecture

The root package (`github.com/wricardo/colorsortgame`) is a library with no
CLI or I/O side effects beyond explicit `Load*`/`Write*` helpers — it can be
imported and driven programmatically. `cmd/colorsort` is the CLI built on
top of it (including `serve`, which starts the `graphqlapi` package's HTTP
server); `cmd/gen` is a standalone level-authoring tool.

## License

MIT

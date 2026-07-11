# colorsortgame

A non-interactive CLI implementation of the water-sort puzzle: pour colored
liquid between tubes until every tube holds a single color (or is empty).

Every command is a single, independent invocation backed by a GraphQL API.
Commands either connect to a remote GraphQL server (pass `--api URL`) or
spawn an ephemeral local server on a random port (default). Games are
stateful and identified by UUID — there's no save file; games are held in
memory and keyed by ID returned from the `new` command.

The bundled 30-level set is compiled into the binary (`go:embed`), so a
single binary runs anywhere with no extra files.

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
./colorsort list                                    # see available levels
./colorsort new --level 1 --json | jq -r '.id'   # start a new game, get its UUID
./colorsort move --game-id <UUID> --from 1 --to 4 # pour tube 1 onto tube 4
./colorsort show --game-id <UUID>                  # reprint the current board
./colorsort undo --game-id <UUID>                  # undo the last move
```

Or connect to a remote server:

```sh
./colorsort --api http://localhost:8080/query new --level 1
./colorsort --api http://localhost:8080/query move --game-id <UUID> --from 1 --to 4
```

## Example: playing level 1

```console
$ ./colorsort new --level 1 --json | jq -r '.id'
8f8f402b-3ba9-4bd9-a3b3-134b8603e808

$ ./colorsort show --game-id 8f8f402b-3ba9-4bd9-a3b3-134b8603e808
Level 1 | moves: 0 | solved: false
 1: [red red green green]
 2: [blue red blue green]
 3: [red blue green blue]
 4: []
 5: []

$ ./colorsort move --game-id 8f8f402b-3ba9-4bd9-a3b3-134b8603e808 --from 1 --to 4
Level 1 | moves: 1 | solved: false
 1: [red red]
 2: [blue red blue green]
 3: [red blue green blue]
 4: [green green]
 5: []

$ ./colorsort move-bulk --game-id 8f8f402b-3ba9-4bd9-a3b3-134b8603e808 --moves "2-4,3-2,3-4,2-3,2-1,3-2,1-3"
Level 1 | moves: 8 | solved: true
 1: []
 2: [blue blue blue blue]
 3: [red red red red]
 4: [green green green green]
 5: []
*** YOU WIN ***
```

Each command is a single GraphQL operation against the API (spawned locally
by default, or pass `--api URL` for a remote server). The game with UUID
`8f8f402b-...` persists across all commands.

## Commands

All commands query a GraphQL API. By default, an ephemeral server is spawned
on a random port; pass `--api URL` to connect to a running server instead.

| Command | Description |
|---|---|
| `list` | List all levels with id, difficulty, tube count, capacity |
| `solvable --level N [--path]` | Check whether level `N` is solvable (exhaustive search); `--path` prints a shortest solution |
| `new --level N` | Start a new game on level `N`; returns game UUID |
| `reset --game-id UUID` | Reset a game to its level's starting state |
| `move --game-id UUID --from A --to B` | Pour tube `A` onto tube `B` (1-indexed) |
| `move-bulk --game-id UUID --moves "1-4,2-3,3-1"` | Apply a comma-separated sequence of moves in order |
| `undo --game-id UUID` | Undo the last move in a game |
| `show` / `status --game-id UUID` | Print the current board |
| `serve --port N` | Start an HTTP server with GraphQL API and web UI (default port 8080) |

Global flags:

| Flag | Description |
|---|---|
| `--api URL` | Connect to a remote GraphQL server (e.g. `--api http://localhost:8080/query`); if omitted, spawn an ephemeral local server |

### `--json`

Every command accepts `--json` to print structured JSON on stdout instead of
the human-readable board/text — useful for scripting. `new` prints the full
game object (including `id`); other game commands (`move`/`undo`/`show`) print
the updated game state; `solvable` prints `{level, solvable, unknown, minMoves, path}`;
`list` prints the level array. Errors are also emitted as JSON
(`{"error": "..."}`) on stderr when `--json` is set. Exit codes are unchanged
either way.

```console
$ ./colorsort new --level 1 --json | jq '.id'
"8f8f402b-3ba9-4bd9-a3b3-134b8603e808"

$ ./colorsort move --game-id 8f8f402b-... --from 1 --to 4 --json | jq '{id, moves, solved}'
{
  "id": "8f8f402b-3ba9-4bd9-a3b3-134b8603e808",
  "moves": 1,
  "solved": false
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

The root package (`github.com/wricardo/colorsortgame`) is a pure library — no
CLI or I/O side effects beyond explicit `Load*`/`Write*` helpers. It can be
imported and driven programmatically (used by the GraphQL resolvers).

`cmd/colorsort` is a non-interactive CLI that executes GraphQL queries/mutations
against a GraphQL API (`graphqlapi` package). It spawns an ephemeral server by
default (using the same handler as `serve`) or connects to a remote server
(`--api URL`). All game state is in memory on the server; there are no local
save files.

`graphqlapi` exports `Handler()` and `Serve()` — the HTTP handler and server
startup function, used by both the `colorsort serve` command and the CLI's
ephemeral server spawning.

`cmd/gen` is a standalone level-authoring tool.

## License

MIT

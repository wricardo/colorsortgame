# CLAUDE.md

Guidance for Claude Code (claude.ai/code) working in this repo.

## Commands

```sh
go build ./cmd/colorsort            # build the CLI binary
go test ./...                       # run all tests (includes CLI integration tests via `go run`)
go test ./... -short                # skip the CLI integration tests (cmd/colorsort/cli_test.go)
go test ./graphqlapi -run TestName  # run a single test
go vet ./...
golangci-lint run ./...
gofmt -s -w .                       # gofmt with simplify; Go Report Card requires -s

mise run serve   # go run ./cmd/colorsort serve
mise run build   # go build -o colorsort ./cmd/colorsort
mise run test    # go vet + golangci-lint
```

Regenerate GraphQL code after edit `graphqlapi/graph/schema.graphqls`:

```sh
cd graphqlapi && go run github.com/99designs/gqlgen generate
```

Verify hand-authored level solvable before add to `levels.json` — see [Level generator](#level-generator) below.

## Architecture

Three layers, each import only one below:

```
cmd/colorsort (CLI, GraphQL client)
        ↓
graphqlapi (GraphQL API + web UI, gqlgen)
        ↓
colorsort (root package: pure game engine library)
```

### Root package (`colorsort`) — pure library

`game.go`, `solve.go`, `levels.go`, `io.go`. No CLI or network code — driven entirely through Go calls (`NewSave`, `ApplyMove`, `Solve`, etc), touches disk only via explicit `Load*`/`Write*` helpers in `io.go`. Foundation for both CLI and GraphQL resolvers.

- **`Save`**: mutable game-state struct (tubes, move count, history, solved/stuck flags). `ApplyMove(from, to)` 1-indexed, mutates in place; recomputes `Solved`/`Stuck` after every move.
- **`Undo`**: no inverse moves stored — reloads level fresh, replays `History[:-1]` from scratch. So `Undo` needs `LevelsPath` resolvable (embedded default if `""`, else real file) even though moves themselves don't need it.
- **`solve.go`**: from-scratch BFS over level's state graph (not reuse `game.go`'s `Save`/`ApplyMove` — parallel unexported funcs `canPourTubes`/`pourTubes`/`isSolvedTubes` operate directly on `[]Tube` for search-loop speed). Two tricks keep it usable on real levels instead of exploding memory:
  - **Canonical state keys** (`tubesKey`): tubes anonymous — swap which index holds empty tube doesn't change puzzle — so states dedup by *sorted* tube contents, not raw tube order.
  - **Parent-pointer path reconstruction**: queue holds only `[]Tube` states, no move lists; shortest path rebuilt at end via `map[key]parentLink`, memory O(1) per visited state instead of O(depth).
  - `SolveStateCap` (20M): safety valve for pathological inputs; hit it → returns `Unknown: true` instead of false "unsolvable".
- **`levels.go`**: embeds `levels.json` via `go:embed` (`DefaultLevels()`), binary needs no external files. `LoadLevelsOrDefault("")` uses embedded set; non-empty path loads + parses external file instead.
- Levels keyed by `id` (not array index), tagged `difficulty` (`easy`/`medium`/`hard`); tubes listed bottom-to-top.

### `graphqlapi` package — GraphQL API + web UI

gqlgen (schema-first). Edit `graph/schema.graphqls`, then regenerate — never hand-edit `graph/generated.go` or `graph/model/models_gen.go`. Resolver *bodies* in `graph/schema.resolvers.go` preserved across regen; `graph/resolver.go` (DI root) and `graph/game_helpers.go` hand-written, generator never touches.

- **State model**: `Resolver` holds in-memory `map[string]*colorsort.Save` behind mutex, keyed by generated game ID (no save-file path here — GraphQL games ephemeral/in-process). `NewResolver()` seeds `levels` from embedded default set only; no GraphQL equiv of CLI's `--levels PATH`.
- **`toModelGame`**: re-looks-up level's difficulty from `r.levels` on every call, since `colorsort.Save` doesn't carry it — quirk to remember if `Save` ever gets serialized/deserialized independently of originating level.
- **`Handler()`** (in `server.go`): builds `http.Handler` (GraphQL endpoint at `/query`, playground at `/playground`, static UI at `/` via `go:embed static`) — factored out so `cmd/colorsort` reuses for its ephemeral local server. `Serve(port)` just wraps `Handler()` with `http.ListenAndServe`.
- Introspection + automatic persisted queries enabled.

### `cmd/colorsort` — CLI, GraphQL client only

CLI does **not** call `colorsort` library directly for game actions — every subcommand (`new`, `move`, `move-bulk`, `undo`, `show`, `reset`, `solvable`, `list`) executes GraphQL query/mutation, uses library package only for parsing (`colorsort.ParseMoves`) and printing (`colorsort.PrintBoard`) client-side.

- **`--api URL`** (global flag, must appear before subcommand) points at running GraphQL server. Omitted → `client.go`'s `New("")` spawns ephemeral server on random port (via `graphqlapi.Handler()`) that lives for single CLI invocation, torn down on exit. So by default every CLI command = fresh empty in-memory game store — `--game-id` values only survive across commands when pointed at same running `--api` server (eg one started with `colorsort serve`).
- No save-file (`--save PATH`) anymore; games identified by UUID returned from `new`, `move`/`undo`/`show`/`reset`/`move-bulk` all require `--game-id`.
- Exit codes: `0` success, `1` error/illegal move, `2` stuck (`move`/`move-bulk`) or solvability unknown (`solvable`).

### `cmd/gen` — level authoring tool (standalone)

Deals shuffled multiset of color segments into filled tubes + empty spares (`-colors`, `-capacity`, `-empty`, `-seed`). **Output not guaranteed solvable** — always verify generated level with solver (eg via `colorsort solvable` against temp levels file, or wire through `Solve` directly) before add to `levels.json`. Every level currently in `levels.json` verified this way; `TestDefaultLevelsAllSolvable` (in `levels_test.go`) re-checks on every test run so bad hand-edit to `levels.json` fails CI instead of ship.

## Testing notes

- `cmd/colorsort/cli_test.go` spawns real subprocesses via `go run ./ ...` (including `go run ./ serve --port N` for background server) — true integration tests, not unit tests, why skippable with `-short`.
- `graphqlapi/server_test.go` starts fresh in-process HTTP server per test (`startTestServer`), tears down via `defer cleanup()` — tests don't share game state.
- `TestDefaultLevelsAllSolvable` + BFS-path-replay tests (`TestSolvePathIsValid`, `TestSolveLevel1EndToEnd`) exist to catch solver or level-data regression producing "solvable" verdict with path that doesn't actually solve board, or a levels.json edit that silently breaks a level.
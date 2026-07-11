# colorsortgame

A non-interactive CLI implementation of the water-sort puzzle: pour colored
liquid between tubes until every tube holds a single color (or is empty).

Every command is a single, independent invocation — no REPL, no long-running
process. Game state lives entirely in a JSON save file, read and rewritten
on each call, so the game can be scripted, tested, or driven by any tool
that can shell out and read JSON.

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

All commands accept `--save PATH` (default `./save.json`); `list`, `solvable`,
`new`, and `reset` also accept `--levels PATH` (default `./levels.json`).

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

`levels.json` bundles all levels; `new`/`reset` load one by `--level` id.
The bundled set (`levels.json` at the repo root) has 30 levels split into
three difficulty tiers:

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

## Architecture

The root package (`github.com/wricardo/colorsortgame`) is a library with no
CLI or I/O side effects beyond explicit `Load*`/`Write*` helpers — it can be
imported and driven programmatically. `cmd/colorsort` is the CLI built on
top of it; `cmd/gen` is a standalone level-authoring tool.

## License

MIT

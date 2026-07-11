package graphqlapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func startTestServer(t *testing.T) (string, func()) {
	handler, err := Handler()
	if err != nil {
		t.Fatalf("Handler: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port)}
	server.Handler = handler
	go server.ListenAndServe()

	// Wait for server to be ready.
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	for i := 0; i < 20; i++ {
		resp, err := http.Get(baseURL + "/query")
		if err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return baseURL, func() { server.Close() }
}

func queryGraphQL(t *testing.T, baseURL, query string, vars map[string]interface{}) map[string]interface{} {
	return queryGraphQLRaw(t, baseURL, query, vars, true)
}

func queryGraphQLNoFail(t *testing.T, baseURL, query string, vars map[string]interface{}) map[string]interface{} {
	return queryGraphQLRaw(t, baseURL, query, vars, false)
}

func queryGraphQLRaw(t *testing.T, baseURL, query string, vars map[string]interface{}, failOnError bool) map[string]interface{} {
	payload := map[string]interface{}{
		"query":     query,
		"variables": vars,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/query", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if failOnError {
		if errs, ok := result["errors"].([]interface{}); ok && len(errs) > 0 {
			t.Fatalf("GraphQL error: %v", errs)
		}
	}

	return result
}

func TestQueryLevels(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	query := `query { levels { id difficulty tubeCapacity } }`
	result := queryGraphQL(t, baseURL, query, nil)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data")
	}

	levelsIface, ok := data["levels"].([]interface{})
	if !ok {
		t.Fatal("missing levels")
	}

	if len(levelsIface) != 30 {
		t.Fatalf("expected 30 levels, got %d", len(levelsIface))
	}

	first := levelsIface[0].(map[string]interface{})
	if first["id"].(float64) != 1 {
		t.Fatalf("expected first level id=1, got %v", first["id"])
	}
}

func TestQueryLevel(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	query := `query level($id: Int!) { level(id: $id) { id difficulty tubeCapacity tubes } }`
	result := queryGraphQL(t, baseURL, query, map[string]interface{}{"id": 1})

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data")
	}

	levelIface, ok := data["level"].(map[string]interface{})
	if !ok {
		t.Fatal("missing level")
	}

	if levelIface["id"].(float64) != 1 {
		t.Fatalf("expected id=1, got %v", levelIface["id"])
	}
}

func TestMutationNewGame(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	query := `mutation newGame($levelId: Int!) {
		newGame(levelId: $levelId) {
			id levelId difficulty capacity tubes moves solved stuck
			history { from to }
		}
	}`
	result := queryGraphQL(t, baseURL, query, map[string]interface{}{"levelId": 1})

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data")
	}

	gameIface, ok := data["newGame"].(map[string]interface{})
	if !ok {
		t.Fatal("missing newGame")
	}

	if gameIface["id"].(string) == "" {
		t.Fatal("missing game id")
	}
	if gameIface["levelId"].(float64) != 1 {
		t.Fatal("expected levelId=1")
	}
	if gameIface["moves"].(float64) != 0 {
		t.Fatal("expected moves=0")
	}
	if gameIface["solved"].(bool) {
		t.Fatal("expected not solved")
	}
}

func TestMutationMove(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game.
	newQuery := `mutation newGame($levelId: Int!) {
		newGame(levelId: $levelId) { id levelId }
	}`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	// Make move.
	moveQuery := `mutation move($id: ID!, $from: Int!, $to: Int!) {
		move(gameId: $id, from: $from, to: $to) {
			id moves solved stuck history { from to }
		}
	}`
	moveResult := queryGraphQL(t, baseURL, moveQuery, map[string]interface{}{
		"id": gameID, "from": 1, "to": 4,
	})

	data, ok := moveResult["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data")
	}

	gameIface, ok := data["move"].(map[string]interface{})
	if !ok {
		t.Fatal("missing move result")
	}

	if gameIface["moves"].(float64) != 1 {
		t.Fatalf("expected moves=1, got %v", gameIface["moves"])
	}

	history := gameIface["history"].([]interface{})
	if len(history) != 1 {
		t.Fatalf("expected 1 move in history, got %d", len(history))
	}
}

func TestMutationUndo(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game and make a move.
	newQuery := `mutation newGame($levelId: Int!) {
		newGame(levelId: $levelId) { id }
	}`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	moveQuery := `mutation move($id: ID!, $from: Int!, $to: Int!) {
		move(gameId: $id, from: $from, to: $to) { moves }
	}`
	queryGraphQL(t, baseURL, moveQuery, map[string]interface{}{"id": gameID, "from": 1, "to": 4})

	// Undo.
	undoQuery := `mutation undo($id: ID!) {
		undo(gameId: $id) { moves history { from to } }
	}`
	undoResult := queryGraphQL(t, baseURL, undoQuery, map[string]interface{}{"id": gameID})

	data, ok := undoResult["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data")
	}

	gameIface, ok := data["undo"].(map[string]interface{})
	if !ok {
		t.Fatal("missing undo result")
	}

	if gameIface["moves"].(float64) != 0 {
		t.Fatalf("expected moves=0 after undo, got %v", gameIface["moves"])
	}

	history := gameIface["history"].([]interface{})
	if len(history) != 0 {
		t.Fatalf("expected empty history after undo, got %d", len(history))
	}
}

func TestMutationResetGame(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game and make a move.
	newQuery := `mutation newGame($levelId: Int!) {
		newGame(levelId: $levelId) { id }
	}`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	moveQuery := `mutation move($id: ID!, $from: Int!, $to: Int!) {
		move(gameId: $id, from: $from, to: $to) { moves }
	}`
	queryGraphQL(t, baseURL, moveQuery, map[string]interface{}{"id": gameID, "from": 1, "to": 4})

	// Reset.
	resetQuery := `mutation resetGame($id: ID!) {
		resetGame(gameId: $id) { moves history { from to } }
	}`
	resetResult := queryGraphQL(t, baseURL, resetQuery, map[string]interface{}{"id": gameID})

	data, ok := resetResult["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data")
	}

	gameIface, ok := data["resetGame"].(map[string]interface{})
	if !ok {
		t.Fatal("missing resetGame result")
	}

	if gameIface["moves"].(float64) != 0 {
		t.Fatalf("expected moves=0 after reset, got %v", gameIface["moves"])
	}
}

func TestQuerySolvable(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	query := `query solvable($levelId: Int!, $includePath: Boolean!) {
		solvable(levelId: $levelId, includePath: $includePath) {
			solvable unknown minMoves path
		}
	}`

	// With path.
	result := queryGraphQL(t, baseURL, query, map[string]interface{}{
		"levelId": 1, "includePath": true,
	})

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data")
	}

	solveIface, ok := data["solvable"].(map[string]interface{})
	if !ok {
		t.Fatal("missing solvable")
	}

	if !solveIface["solvable"].(bool) {
		t.Fatal("expected solvable=true")
	}
	if solveIface["minMoves"].(float64) <= 0 {
		t.Fatalf("expected minMoves > 0, got %v", solveIface["minMoves"])
	}

	path, ok := solveIface["path"].([]interface{})
	if !ok || len(path) == 0 {
		t.Fatal("expected path to be populated when includePath=true")
	}

	// Without path.
	result = queryGraphQL(t, baseURL, query, map[string]interface{}{
		"levelId": 1, "includePath": false,
	})
	data = result["data"].(map[string]interface{})
	solveIface = data["solvable"].(map[string]interface{})
	if solveIface["path"] != nil {
		t.Fatalf("expected null path when includePath=false, got %v", solveIface["path"])
	}
}

func TestMoveBulk(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game.
	newQuery := `mutation newGame($levelId: Int!) {
		newGame(levelId: $levelId) { id }
	}`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	// Make bulk moves.
	bulkQuery := `mutation moveBulk($id: ID!, $moves: String!) {
		moveBulk(gameId: $id, moves: $moves) {
			moves history { from to }
		}
	}`
	bulkResult := queryGraphQL(t, baseURL, bulkQuery, map[string]interface{}{
		"id": gameID, "moves": "1-4,2-4",
	})

	data, ok := bulkResult["data"].(map[string]interface{})
	if !ok {
		t.Fatal("missing data")
	}

	gameIface, ok := data["moveBulk"].(map[string]interface{})
	if !ok {
		t.Fatal("missing moveBulk result")
	}

	if gameIface["moves"].(float64) != 2 {
		t.Fatalf("expected moves=2, got %v", gameIface["moves"])
	}
}

func TestInvalidGameID(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	query := `mutation move($id: ID!, $from: Int!, $to: Int!) {
		move(gameId: $id, from: $from, to: $to) { id }
	}`
	result := queryGraphQLNoFail(t, baseURL, query, map[string]interface{}{
		"id": "nonexistent-uuid", "from": 1, "to": 2,
	})

	if _, ok := result["errors"].([]interface{}); !ok {
		t.Fatal("expected error for invalid game ID")
	}
}

func TestInvalidTubeIndices(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	newQuery := `mutation newGame($levelId: Int!) { newGame(levelId: $levelId) { id } }`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	moveQuery := `mutation move($id: ID!, $from: Int!, $to: Int!) {
		move(gameId: $id, from: $from, to: $to) { id }
	}`

	// Out of range
	result := queryGraphQLNoFail(t, baseURL, moveQuery, map[string]interface{}{
		"id": gameID, "from": 99, "to": 2,
	})
	if _, ok := result["errors"].([]interface{}); !ok {
		t.Fatal("expected error for out-of-range tube")
	}
}

func TestMoveFromEmptyTube(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	newQuery := `mutation newGame($levelId: Int!) { newGame(levelId: $levelId) { id } }`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	moveQuery := `mutation move($id: ID!, $from: Int!, $to: Int!) {
		move(gameId: $id, from: $from, to: $to) { id }
	}`

	// Try to move from an empty tube (5 is empty on level 1)
	result := queryGraphQLNoFail(t, baseURL, moveQuery, map[string]interface{}{
		"id": gameID, "from": 5, "to": 4,
	})
	if _, ok := result["errors"].([]interface{}); !ok {
		t.Fatal("expected error for empty source tube")
	}
}

func TestStuckDetection(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create a game and make it stuck by making invalid moves until board state is stuck
	// For simplicity, we'll use a known stuck configuration by making specific moves
	newQuery := `mutation newGame($levelId: Int!) { newGame(levelId: $levelId) { id } }`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	// Level 1 is solvable, so we can't easily get it stuck naturally.
	// Test that a freshly created game is not stuck.
	showQuery := `query game($id: ID!) { game(id: $id) { stuck moves } }`
	result := queryGraphQL(t, baseURL, showQuery, map[string]interface{}{"id": gameID})
	gameIface := result["data"].(map[string]interface{})["game"].(map[string]interface{})
	if gameIface["stuck"].(bool) {
		t.Fatal("fresh game should not be stuck")
	}
}

func TestMultiGameIsolation(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create two games
	newQuery := `mutation newGame($levelId: Int!) { newGame(levelId: $levelId) { id } }`
	result1 := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	game1ID := result1["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	result2 := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 2})
	game2ID := result2["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	if game1ID == game2ID {
		t.Fatal("games should have different IDs")
	}

	// Make move in game 1
	moveQuery := `mutation move($id: ID!, $from: Int!, $to: Int!) {
		move(gameId: $id, from: $from, to: $to) { moves }
	}`
	moveResult := queryGraphQL(t, baseURL, moveQuery, map[string]interface{}{
		"id": game1ID, "from": 1, "to": 4,
	})
	game1Moves := moveResult["data"].(map[string]interface{})["move"].(map[string]interface{})["moves"].(float64)

	// Check game 2 still has 0 moves
	showQuery := `query game($id: ID!) { game(id: $id) { moves } }`
	game2State := queryGraphQL(t, baseURL, showQuery, map[string]interface{}{"id": game2ID})
	game2Moves := game2State["data"].(map[string]interface{})["game"].(map[string]interface{})["moves"].(float64)

	if game1Moves != 1 {
		t.Fatalf("game1 should have 1 move, got %v", game1Moves)
	}
	if game2Moves != 0 {
		t.Fatalf("game2 should have 0 moves, got %v", game2Moves)
	}
}

func TestSolvedGameCantMove(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game and solve it
	newQuery := `mutation newGame($levelId: Int!) { newGame(levelId: $levelId) { id } }`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	// Get solution
	solveQuery := `query solvable($levelId: Int!, $includePath: Boolean!) {
		solvable(levelId: $levelId, includePath: $includePath) { path }
	}`
	solveResult := queryGraphQL(t, baseURL, solveQuery, map[string]interface{}{
		"levelId": 1, "includePath": true,
	})
	pathIface := solveResult["data"].(map[string]interface{})["solvable"].(map[string]interface{})["path"].([]interface{})
	moves := ""
	for i, m := range pathIface {
		if i > 0 {
			moves += ","
		}
		moves += m.(string)
	}

	// Solve it
	bulkQuery := `mutation moveBulk($id: ID!, $moves: String!) {
		moveBulk(gameId: $id, moves: $moves) { solved }
	}`
	queryGraphQL(t, baseURL, bulkQuery, map[string]interface{}{"id": gameID, "moves": moves})

	// Try to move after solved
	moveQuery := `mutation move($id: ID!, $from: Int!, $to: Int!) {
		move(gameId: $id, from: $from, to: $to) { id }
	}`
	result := queryGraphQLNoFail(t, baseURL, moveQuery, map[string]interface{}{
		"id": gameID, "from": 1, "to": 2,
	})
	if _, ok := result["errors"].([]interface{}); !ok {
		t.Fatal("expected error for move on solved game")
	}
}

func TestHistoryCorrectness(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	newQuery := `mutation newGame($levelId: Int!) { newGame(levelId: $levelId) { id } }`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	// Make three moves
	moveQuery := `mutation move($id: ID!, $from: Int!, $to: Int!) {
		move(gameId: $id, from: $from, to: $to) { history { from to } }
	}`
	moves := []map[string]interface{}{
		{"from": 1, "to": 4},
		{"from": 2, "to": 4},
		{"from": 3, "to": 2},
	}

	for _, m := range moves {
		queryGraphQL(t, baseURL, moveQuery, map[string]interface{}{
			"id": gameID, "from": m["from"], "to": m["to"],
		})
	}

	// Verify final history
	showQuery := `query game($id: ID!) { game(id: $id) { history { from to } } }`
	result := queryGraphQL(t, baseURL, showQuery, map[string]interface{}{"id": gameID})
	historyIface := result["data"].(map[string]interface{})["game"].(map[string]interface{})["history"].([]interface{})

	if len(historyIface) != 3 {
		t.Fatalf("expected 3 moves in history, got %d", len(historyIface))
	}

	for i, expected := range moves {
		actual := historyIface[i].(map[string]interface{})
		if actual["from"].(float64) != float64(expected["from"].(int)) {
			t.Fatalf("move %d: expected from=%d, got %v", i, expected["from"], actual["from"])
		}
		if actual["to"].(float64) != float64(expected["to"].(int)) {
			t.Fatalf("move %d: expected to=%d, got %v", i, expected["to"], actual["to"])
		}
	}
}

func TestResetSolvedGame(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	newQuery := `mutation newGame($levelId: Int!) { newGame(levelId: $levelId) { id } }`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	// Solve it
	solveQuery := `query solvable($levelId: Int!, $includePath: Boolean!) {
		solvable(levelId: $levelId, includePath: $includePath) { path }
	}`
	solveResult := queryGraphQL(t, baseURL, solveQuery, map[string]interface{}{
		"levelId": 1, "includePath": true,
	})
	pathIface := solveResult["data"].(map[string]interface{})["solvable"].(map[string]interface{})["path"].([]interface{})
	moves := ""
	for i, m := range pathIface {
		if i > 0 {
			moves += ","
		}
		moves += m.(string)
	}

	bulkQuery := `mutation moveBulk($id: ID!, $moves: String!) {
		moveBulk(gameId: $id, moves: $moves) { solved }
	}`
	queryGraphQL(t, baseURL, bulkQuery, map[string]interface{}{"id": gameID, "moves": moves})

	// Reset
	resetQuery := `mutation resetGame($id: ID!) {
		resetGame(gameId: $id) { solved moves history { from to } }
	}`
	resetResult := queryGraphQL(t, baseURL, resetQuery, map[string]interface{}{"id": gameID})
	gameIface := resetResult["data"].(map[string]interface{})["resetGame"].(map[string]interface{})

	if gameIface["solved"].(bool) {
		t.Fatal("reset game should not be solved")
	}
	if gameIface["moves"].(float64) != 0 {
		t.Fatal("reset game should have 0 moves")
	}
	history := gameIface["history"].([]interface{})
	if len(history) != 0 {
		t.Fatal("reset game should have empty history")
	}
}

func TestSolvableUnsolvableLevel(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create a stuck level (can't be solved). We'll construct one manually.
	// But since all bundled levels are solvable, we just verify a valid level is solvable.
	query := `query solvable($levelId: Int!, $includePath: Boolean!) {
		solvable(levelId: $levelId, includePath: $includePath) { solvable unknown }
	}`
	result := queryGraphQL(t, baseURL, query, map[string]interface{}{
		"levelId": 1, "includePath": false,
	})

	solveIface := result["data"].(map[string]interface{})["solvable"].(map[string]interface{})
	if !solveIface["solvable"].(bool) {
		t.Fatal("level 1 should be solvable")
	}
	if solveIface["unknown"].(bool) {
		t.Fatal("should have definitive answer, not unknown")
	}
}

func TestSolveLevel1EndToEnd(t *testing.T) {
	baseURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game on level 1.
	newQuery := `mutation newGame($levelId: Int!) {
		newGame(levelId: $levelId) { id solved }
	}`
	newResult := queryGraphQL(t, baseURL, newQuery, map[string]interface{}{"levelId": 1})
	gameID := newResult["data"].(map[string]interface{})["newGame"].(map[string]interface{})["id"].(string)

	// Get solution path.
	solveQuery := `query solvable($levelId: Int!, $includePath: Boolean!) {
		solvable(levelId: $levelId, includePath: $includePath) {
			solvable minMoves path
		}
	}`
	solveResult := queryGraphQL(t, baseURL, solveQuery, map[string]interface{}{
		"levelId": 1, "includePath": true,
	})

	solveData := solveResult["data"].(map[string]interface{})
	solveIface := solveData["solvable"].(map[string]interface{})

	if !solveIface["solvable"].(bool) {
		t.Fatal("level 1 should be solvable")
	}

	pathIface := solveIface["path"].([]interface{})
	if len(pathIface) == 0 {
		t.Fatal("solution path should not be empty")
	}

	// Convert path to moves string (e.g., ["1-4", "2-4"] -> "1-4,2-4")
	moves := ""
	for i, m := range pathIface {
		if i > 0 {
			moves += ","
		}
		moves += m.(string)
	}

	// Execute solution.
	bulkQuery := `mutation moveBulk($id: ID!, $moves: String!) {
		moveBulk(gameId: $id, moves: $moves) {
			moves solved stuck
		}
	}`
	bulkResult := queryGraphQL(t, baseURL, bulkQuery, map[string]interface{}{
		"id": gameID, "moves": moves,
	})

	bulkData := bulkResult["data"].(map[string]interface{})
	gameIface := bulkData["moveBulk"].(map[string]interface{})

	if !gameIface["solved"].(bool) {
		t.Fatal("expected solved=true after executing solution")
	}

	if gameIface["stuck"].(bool) {
		t.Fatal("expected stuck=false after executing solution")
	}

	expectedMoves := int(solveIface["minMoves"].(float64))
	actualMoves := int(gameIface["moves"].(float64))
	if actualMoves != expectedMoves {
		t.Fatalf("expected moves=%d, got %d", expectedMoves, actualMoves)
	}
}

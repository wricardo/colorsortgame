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

	if errs, ok := result["errors"].([]interface{}); ok && len(errs) > 0 {
		t.Fatalf("GraphQL error: %v", errs)
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

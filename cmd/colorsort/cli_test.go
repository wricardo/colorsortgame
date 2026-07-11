package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"
)

func startTestServer(t *testing.T) (string, func()) {
	// Get random port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	// Start server via go run
	cmd := exec.Command("go", "run", "./", "serve", "--port", fmt.Sprintf("%d", port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start server: %v", err)
	}

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	return fmt.Sprintf("http://127.0.0.1:%d/query", port), func() {
		cmd.Process.Kill()
		cmd.Wait()
	}
}

func runCLI(t *testing.T, args ...string) (string, error) {
	allArgs := append([]string{"run", "./"}, args...)
	cmd := exec.Command("go", allArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestCLIList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	out, err := runCLI(t, "list")
	if err != nil {
		t.Fatalf("colorsort list: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected output from list command")
	}
}

func TestCLIListJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	out, err := runCLI(t, "list", "--json")
	if err != nil {
		t.Fatalf("colorsort list --json: %v", err)
	}

	var levels []interface{}
	if err := json.Unmarshal([]byte(out), &levels); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}

	if len(levels) != 30 {
		t.Fatalf("expected 30 levels, got %d", len(levels))
	}
}

func TestCLISolvable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	out, err := runCLI(t, "solvable", "--level", "1")
	if err != nil {
		t.Fatalf("colorsort solvable: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected output from solvable command")
	}
}


func TestCLINewAndMove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	// Test with remote API
	apiURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game
	out, err := runCLI(t, "--api", apiURL, "new", "--level", "1", "--json")
	if err != nil {
		t.Fatalf("colorsort new: %v\nout: %s", err, out)
	}

	var game map[string]interface{}
	if err := json.Unmarshal([]byte(out), &game); err != nil {
		t.Fatalf("unmarshal game JSON: %v\nout: %s", err, out)
	}

	gameID, ok := game["id"].(string)
	if !ok || gameID == "" {
		t.Fatal("expected game id in response")
	}

	// Make a move
	out, err = runCLI(t, "--api", apiURL, "move", "--game-id", gameID, "--from", "1", "--to", "4")
	if err != nil {
		t.Fatalf("colorsort move: %v\nout: %s", err, out)
	}

	if len(out) == 0 {
		t.Fatal("expected output from move command")
	}
}

func TestCLIShowStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	apiURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game
	out, err := runCLI(t, "--api", apiURL, "new", "--level", "1", "--json")
	if err != nil {
		t.Fatalf("colorsort new: %v", err)
	}

	var game map[string]interface{}
	json.Unmarshal([]byte(out), &game)
	gameID := game["id"].(string)

	// Show status
	out, err = runCLI(t, "--api", apiURL, "show", "--game-id", gameID)
	if err != nil {
		t.Fatalf("colorsort show: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected output from show command")
	}
}

func TestCLIUndo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	apiURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game and make a move
	out, err := runCLI(t, "--api", apiURL, "new", "--level", "1", "--json")
	if err != nil {
		t.Fatalf("colorsort new: %v", err)
	}

	var game map[string]interface{}
	json.Unmarshal([]byte(out), &game)
	gameID := game["id"].(string)

	// Move
	runCLI(t, "--api", apiURL, "move", "--game-id", gameID, "--from", "1", "--to", "4")

	// Undo
	out, err = runCLI(t, "--api", apiURL, "undo", "--game-id", gameID)
	if err != nil {
		t.Fatalf("colorsort undo: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected output from undo command")
	}
}

func TestCLIMoveBulk(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	apiURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game
	out, err := runCLI(t, "--api", apiURL, "new", "--level", "1", "--json")
	if err != nil {
		t.Fatalf("colorsort new: %v", err)
	}

	var game map[string]interface{}
	json.Unmarshal([]byte(out), &game)
	gameID := game["id"].(string)

	// Make multiple moves
	out, err = runCLI(t, "--api", apiURL, "move-bulk", "--game-id", gameID, "--moves", "1-4,2-4")
	if err != nil {
		t.Fatalf("colorsort move-bulk: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected output from move-bulk command")
	}
}

func TestCLIReset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	apiURL, cleanup := startTestServer(t)
	defer cleanup()

	// Create game and make a move
	out, err := runCLI(t, "--api", apiURL, "new", "--level", "1", "--json")
	if err != nil {
		t.Fatalf("colorsort new: %v", err)
	}

	var game map[string]interface{}
	json.Unmarshal([]byte(out), &game)
	gameID := game["id"].(string)

	// Move
	runCLI(t, "--api", apiURL, "move", "--game-id", gameID, "--from", "1", "--to", "4")

	// Reset
	out, err = runCLI(t, "--api", apiURL, "reset", "--game-id", gameID)
	if err != nil {
		t.Fatalf("colorsort reset: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected output from reset command")
	}
}

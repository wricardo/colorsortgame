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

func getColorsortBinary(t *testing.T) string {
	// Get the directory of the test file
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	// Try to find the colorsort binary
	paths := []string{
		wd + "/colorsort",
		"colorsort",
		"./colorsort",
	}

	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}

	// If not found, try building it in the current directory
	cmd := exec.Command("go", "build", "-o", wd+"/colorsort", "./...")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build colorsort: %v", err)
	}
	return wd + "/colorsort"
}

func startTestServer(t *testing.T, binary string) (string, func()) {
	// Get random port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	// Start server
	cmd := exec.Command(binary, "serve", "--port", fmt.Sprintf("%d", port))
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

func runCLI(t *testing.T, binary string, args ...string) (string, error) {
	cmd := exec.Command(binary, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestCLIList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	binary := getColorsortBinary(t)
	out, err := runCLI(t, binary, "list")
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

	binary := getColorsortBinary(t)
	out, err := runCLI(t, binary, "list", "--json")
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

	binary := getColorsortBinary(t)
	out, err := runCLI(t, binary, "solvable", "--level", "1")
	if err != nil {
		t.Fatalf("colorsort solvable: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected output from solvable command")
	}
}

func TestCLISolvableWithPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	binary := getColorsortBinary(t)
	out, err := runCLI(t, binary, "solvable", "--level", "1", "--path")
	if err != nil {
		t.Fatalf("colorsort solvable --path: %v", err)
	}

	// Should contain comma-separated moves
	if len(out) == 0 {
		t.Fatal("expected output from solvable command")
	}
}

func TestCLINewAndMove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI test in short mode")
	}

	binary := getColorsortBinary(t)
	// Test with remote API
	apiURL, cleanup := startTestServer(t, binary)
	defer cleanup()

	// Create game
	out, err := runCLI(t, binary, "--api", apiURL, "new", "--level", "1", "--json")
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
	out, err = runCLI(t, binary, "--api", apiURL, "move", "--game-id", gameID, "--from", "1", "--to", "4")
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

	binary := getColorsortBinary(t)
	apiURL, cleanup := startTestServer(t, binary)
	defer cleanup()

	// Create game
	out, err := runCLI(t, binary, "--api", apiURL, "new", "--level", "1", "--json")
	if err != nil {
		t.Fatalf("colorsort new: %v", err)
	}

	var game map[string]interface{}
	json.Unmarshal([]byte(out), &game)
	gameID := game["id"].(string)

	// Show status
	out, err = runCLI(t, binary, "--api", apiURL, "show", "--game-id", gameID)
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

	binary := getColorsortBinary(t)
	apiURL, cleanup := startTestServer(t, binary)
	defer cleanup()

	// Create game and make a move
	out, err := runCLI(t, binary, "--api", apiURL, "new", "--level", "1", "--json")
	if err != nil {
		t.Fatalf("colorsort new: %v", err)
	}

	var game map[string]interface{}
	json.Unmarshal([]byte(out), &game)
	gameID := game["id"].(string)

	// Move
	runCLI(t, binary, "--api", apiURL, "move", "--game-id", gameID, "--from", "1", "--to", "4")

	// Undo
	out, err = runCLI(t, binary, "--api", apiURL, "undo", "--game-id", gameID)
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

	binary := getColorsortBinary(t)
	apiURL, cleanup := startTestServer(t, binary)
	defer cleanup()

	// Create game
	out, err := runCLI(t, binary, "--api", apiURL, "new", "--level", "1", "--json")
	if err != nil {
		t.Fatalf("colorsort new: %v", err)
	}

	var game map[string]interface{}
	json.Unmarshal([]byte(out), &game)
	gameID := game["id"].(string)

	// Make multiple moves
	out, err = runCLI(t, binary, "--api", apiURL, "move-bulk", "--game-id", gameID, "--moves", "1-4,2-4")
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

	binary := getColorsortBinary(t)
	apiURL, cleanup := startTestServer(t, binary)
	defer cleanup()

	// Create game and make a move
	out, err := runCLI(t, binary, "--api", apiURL, "new", "--level", "1", "--json")
	if err != nil {
		t.Fatalf("colorsort new: %v", err)
	}

	var game map[string]interface{}
	json.Unmarshal([]byte(out), &game)
	gameID := game["id"].(string)

	// Move
	runCLI(t, binary, "--api", apiURL, "move", "--game-id", gameID, "--from", "1", "--to", "4")

	// Reset
	out, err = runCLI(t, binary, "--api", apiURL, "reset", "--game-id", gameID)
	if err != nil {
		t.Fatalf("colorsort reset: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected output from reset command")
	}
}

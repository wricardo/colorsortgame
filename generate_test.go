package colorsort

import "testing"

func TestGenerate(t *testing.T) {
	cfg := GenerateConfig{
		Difficulty: "easy",
		Colors:     3,
		Capacity:   4,
		Empty:      2,
		Seed:       42,
	}
	lvl, err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if lvl.Difficulty != "easy" {
		t.Fatalf("expected difficulty=easy, got %s", lvl.Difficulty)
	}
	if lvl.Capacity != 4 {
		t.Fatalf("expected capacity=4, got %d", lvl.Capacity)
	}
	if len(lvl.Tubes) != 5 {
		t.Fatalf("expected 5 tubes (3 colors + 2 empty), got %d", len(lvl.Tubes))
	}

	// Check filled tubes have correct capacity
	for i := 0; i < 3; i++ {
		if len(lvl.Tubes[i]) != 4 {
			t.Fatalf("tube %d: expected 4 segments, got %d", i, len(lvl.Tubes[i]))
		}
	}

	// Check empty tubes are empty
	for i := 3; i < 5; i++ {
		if len(lvl.Tubes[i]) != 0 {
			t.Fatalf("tube %d: expected empty, got %d segments", i, len(lvl.Tubes[i]))
		}
	}

	// Same seed produces same result
	lvl2, _ := Generate(cfg)
	for i := range lvl.Tubes {
		for j := range lvl.Tubes[i] {
			if lvl.Tubes[i][j] != lvl2.Tubes[i][j] {
				t.Fatal("same seed should produce same puzzle")
			}
		}
	}
}

func TestGenerateErrors(t *testing.T) {
	tests := []struct {
		name string
		cfg  GenerateConfig
	}{
		{"too many colors", GenerateConfig{Colors: 20, Capacity: 4, Empty: 2}},
		{"zero colors", GenerateConfig{Colors: 0, Capacity: 4, Empty: 2}},
		{"zero capacity", GenerateConfig{Colors: 3, Capacity: 0, Empty: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Generate(tt.cfg)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

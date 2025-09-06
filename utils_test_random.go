package main

import (
	"testing"
)

func TestCheckRandomChance(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid 0%", "0", false},
		{"valid 100%", "100", false},
		{"valid 50%", "50", false},
		{"invalid negative", "-1", true},
		{"invalid over 100", "101", true},
		{"invalid non-integer", "abc", true},
		{"invalid empty", "", true},
		{"invalid float", "50.5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := checkRandomChance(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkRandomChance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCheckRandomChanceValues tests specific probability values
func TestCheckRandomChanceValues(t *testing.T) {
	// Test 0% - should always return false
	for i := 0; i < 10; i++ {
		result, err := checkRandomChance("0")
		if err != nil {
			t.Errorf("checkRandomChance(\"0\") returned error: %v", err)
		}
		if result {
			t.Error("checkRandomChance(\"0\") should always return false")
		}
	}

	// Test 100% - should always return true
	for i := 0; i < 10; i++ {
		result, err := checkRandomChance("100")
		if err != nil {
			t.Errorf("checkRandomChance(\"100\") returned error: %v", err)
		}
		if !result {
			t.Error("checkRandomChance(\"100\") should always return true")
		}
	}
}

// TestCheckRandomChanceDistribution tests the statistical distribution
func TestCheckRandomChanceDistribution(t *testing.T) {
	// Skip this test in short mode as it takes time
	if testing.Short() {
		t.Skip("skipping statistical test in short mode")
	}

	tests := []struct {
		probability string
		expected    float64
		tolerance   float64
	}{
		{"10", 0.10, 0.03}, // 10% ± 3%
		{"25", 0.25, 0.05}, // 25% ± 5%
		{"50", 0.50, 0.05}, // 50% ± 5%
		{"75", 0.75, 0.05}, // 75% ± 5%
		{"90", 0.90, 0.03}, // 90% ± 3%
	}

	for _, tt := range tests {
		t.Run("probability_"+tt.probability, func(t *testing.T) {
			const iterations = 1000
			trueCount := 0

			for i := 0; i < iterations; i++ {
				result, err := checkRandomChance(tt.probability)
				if err != nil {
					t.Fatalf("checkRandomChance(%s) returned error: %v", tt.probability, err)
				}
				if result {
					trueCount++
				}
			}

			actual := float64(trueCount) / float64(iterations)
			if actual < tt.expected-tt.tolerance || actual > tt.expected+tt.tolerance {
				t.Errorf("Expected ~%.2f for %s%% probability, got %.2f (%.1f%% of %d iterations)",
					tt.expected, tt.probability, actual, actual*100, iterations)
			}
		})
	}
}

// TestCheckRandomChanceErrorMessages tests error message content
func TestCheckRandomChanceErrorMessages(t *testing.T) {
	tests := []struct {
		value       string
		expectedErr string
	}{
		{"-1", "invalid random_chance value: -1 (must be 0-100)"},
		{"101", "invalid random_chance value: 101 (must be 0-100)"},
		{"abc", "invalid random_chance value: abc (must be integer)"},
		{"", "invalid random_chance value:  (must be integer)"},
	}

	for _, tt := range tests {
		t.Run("error_"+tt.value, func(t *testing.T) {
			_, err := checkRandomChance(tt.value)
			if err == nil {
				t.Errorf("checkRandomChance(%s) should return error", tt.value)
				return
			}
			if err.Error() != tt.expectedErr {
				t.Errorf("checkRandomChance(%s) error = %q, want %q",
					tt.value, err.Error(), tt.expectedErr)
			}
		})
	}
}

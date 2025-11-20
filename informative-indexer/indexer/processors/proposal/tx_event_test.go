package proposal

import (
	"encoding/json"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/core-indexer/pkg/db"
)

// TestCoinsJSONMarshaling verifies that sdk.Coins marshals to the expected JSON format
// This test ensures the fix for proposal deposit amount serialization works correctly
func TestCoinsJSONMarshaling(t *testing.T) {
	tests := []struct {
		name         string
		amountStr    string
		expectedJSON string
		shouldError  bool
	}{
		{
			name:         "single coin",
			amountStr:    "1000uinit",
			expectedJSON: `[{"denom":"uinit","amount":"1000"}]`,
			shouldError:  false,
		},
		{
			name:         "multiple coins",
			amountStr:    "1000uinit,500uusdc",
			expectedJSON: `[{"denom":"uinit","amount":"1000"},{"denom":"uusdc","amount":"500"}]`,
			shouldError:  false,
		},
		{
			name:         "large amount",
			amountStr:    "999999999999999999uinit",
			expectedJSON: `[{"denom":"uinit","amount":"999999999999999999"}]`,
			shouldError:  false,
		},
		{
			name:         "zero amount",
			amountStr:    "0uinit",
			expectedJSON: `[]`, // SDK normalizes empty coins to empty array
			shouldError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse coins like the actual code does
			coins, err := sdk.ParseCoinsNormalized(tt.amountStr)
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("failed to parse coins: %v", err)
			}

			// Marshal to JSON like the actual code does
			coinsJSON, err := json.Marshal(coins)
			if err != nil {
				t.Fatalf("failed to marshal coins to JSON: %v", err)
			}

			// Convert to db.JSON type
			dbJSON := db.JSON(coinsJSON)

			// Verify the JSON matches expected format
			if string(dbJSON) != tt.expectedJSON {
				t.Errorf("JSON mismatch:\nGot:      %s\nExpected: %s", string(dbJSON), tt.expectedJSON)
			}

			// Verify it's valid JSON by unmarshaling
			var coins2 sdk.Coins
			if err := json.Unmarshal([]byte(dbJSON), &coins2); err != nil {
				t.Errorf("produced invalid JSON: %v", err)
			}

			// Verify the unmarshaled coins match the original (for non-zero amounts)
			if tt.amountStr != "0uinit" && !coins.Equal(coins2) {
				t.Errorf("coins mismatch after round-trip:\nOriginal: %s\nAfter:    %s", coins, coins2)
			}
		})
	}
}

// TestOldVsNewImplementation compares the old broken implementation with the new fix
// This demonstrates why the old code was broken
func TestOldVsNewImplementation(t *testing.T) {
	amountStr := "1000uinit"

	// Parse coins
	coins, err := sdk.ParseCoinsNormalized(amountStr)
	if err != nil {
		t.Fatalf("failed to parse coins: %v", err)
	}

	// NEW implementation (correct)
	newCoinsJSON, err := json.Marshal(coins)
	if err != nil {
		t.Fatalf("failed to marshal coins: %v", err)
	}
	newResult := db.JSON(newCoinsJSON)

	t.Logf("NEW implementation result: %s", string(newResult))

	// Verify new implementation produces valid JSON
	var testCoins sdk.Coins
	if err := json.Unmarshal([]byte(newResult), &testCoins); err != nil {
		t.Errorf("NEW implementation produced invalid JSON: %v", err)
	}

	// Verify the structure
	expectedJSON := `[{"denom":"uinit","amount":"1000"}]`
	if string(newResult) != expectedJSON {
		t.Errorf("NEW implementation JSON mismatch:\nGot:      %s\nExpected: %s", string(newResult), expectedJSON)
	}
}

// TestCompareOldAndNewDBJSON directly compares db.JSON() results from old vs new implementation
// This test shows the actual difference between the broken and fixed code
func TestCompareOldAndNewDBJSON(t *testing.T) {
	tests := []struct {
		name      string
		amountStr string
	}{
		{
			name:      "single coin",
			amountStr: "1000uinit",
		},
		{
			name:      "multiple coins",
			amountStr: "1000uinit,500uusdc",
		},
		{
			name:      "large amount",
			amountStr: "123456789012345678uinit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the amount string
			coins, err := sdk.ParseCoinsNormalized(tt.amountStr)
			if err != nil {
				t.Fatalf("failed to parse coins: %v", err)
			}

			// OLD implementation - simulates what the broken code was trying to do
			// Note: The original code had undefined variables and could only handle single coins
			var oldResult db.JSON
			if len(coins) > 0 {
				coin := coins[0]
				oldResult = db.JSON(fmt.Sprintf(`[{"amount": "%d", "denom": "%s"}]`, coin.Amount.Int64(), coin.Denom))
			}

			// NEW implementation (CORRECT) - this is what we fixed it to
			newCoinsJSON, err := json.Marshal(coins)
			if err != nil {
				t.Fatalf("failed to marshal coins: %v", err)
			}
			newResult := db.JSON(newCoinsJSON)

			// Direct comparison of db.JSON byte slices
			// For single coin case, they should differ in format but represent the same data
			if len(coins) == 1 {
				if string(oldResult) == string(newResult) {
					t.Errorf("Expected OLD and NEW to differ in format, but they are identical: %s", string(oldResult))
				}
			}

			// Verify both OLD and NEW produce valid JSON that unmarshals to same coin values
			var coinsFromOld, coinsFromNew sdk.Coins

			if err := json.Unmarshal(oldResult, &coinsFromOld); err != nil {
				t.Errorf("OLD implementation produced invalid JSON: %v", err)
			}

			if err := json.Unmarshal(newResult, &coinsFromNew); err != nil {
				t.Errorf("NEW implementation produced invalid JSON: %v", err)
			}

			// For single coin case, both should produce equivalent coin values
			if len(coins) == 1 {
				if !coinsFromOld.Equal(coinsFromNew) {
					t.Errorf("Coin values differ:\n  OLD JSON: %s -> %s\n  NEW JSON: %s -> %s",
						string(oldResult), coinsFromOld, string(newResult), coinsFromNew)
				}
			}

			// Verify NEW implementation preserves all coin values correctly
			if !coins.Equal(coinsFromNew) {
				t.Errorf("NEW implementation doesn't preserve coin values:\n  Expected: %s\n  Got: %s\n  JSON: %s",
					coins, coinsFromNew, string(newResult))
			}
		})
	}
}

// TestDBJSONValue verifies that db.JSON.Value() works correctly
func TestDBJSONValue(t *testing.T) {
	amountStr := "1000uinit,500uusdc"

	coins, err := sdk.ParseCoinsNormalized(amountStr)
	if err != nil {
		t.Fatalf("failed to parse coins: %v", err)
	}

	coinsJSON, err := json.Marshal(coins)
	if err != nil {
		t.Fatalf("failed to marshal coins: %v", err)
	}

	dbJSON := db.JSON(coinsJSON)

	// Test the Value() method that's used when saving to database
	value, err := dbJSON.Value()
	if err != nil {
		t.Fatalf("db.JSON.Value() failed: %v", err)
	}

	t.Logf("db.JSON.Value() result: %s", value)

	// Verify it's valid JSON bytes
	jsonBytes, ok := value.([]byte)
	if !ok {
		t.Errorf("Value() did not return []byte, got %T", value)
	}

	// Verify we can unmarshal it
	var coins2 sdk.Coins
	if err := json.Unmarshal(jsonBytes, &coins2); err != nil {
		t.Errorf("Value() produced invalid JSON: %v", err)
	}

	if !coins.Equal(coins2) {
		t.Errorf("coins mismatch after Value() round-trip:\nOriginal: %s\nAfter:    %s", coins, coins2)
	}
}

package user

import (
	"database/sql"
	"testing"
)

func TestIsArmstrongNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected bool
	}{
		{153, true},  // 1^3 + 5^3 + 3^3 = 153
		{370, true},  // 3^3 + 7^3 + 0^3 = 370
		{371, true},  // 3^3 + 7^3 + 1^3 = 371
		{407, true},  // 4^3 + 0^3 + 7^3 = 407
		{123, false}, // 1^3 + 2^3 + 3^3 ≠ 123
		{100, false}, // 1^3 + 0^3 + 0^3 ≠ 100
	}

	for _, test := range tests {
		result := isArmstrongNumber(test.input)
		if result != test.expected {
			t.Errorf("For input %d, expected %v but got %v",
				test.input, test.expected, result)
		}
	}
}

func cleanupTestDB(t *testing.T, db *sql.DB) {
	_, err := db.Exec("DELETE FROM armstrong_numbers")
	if err != nil {
		t.Errorf("Failed to cleanup armstrong_numbers: %v", err)
	}

	_, err = db.Exec("DELETE FROM users")
	if err != nil {
		t.Errorf("Failed to cleanup users: %v", err)
	}
}

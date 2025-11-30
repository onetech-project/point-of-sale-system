package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T001: Example unit test to verify test framework setup
func TestExampleUnit(t *testing.T) {
	// Arrange
	expected := "test"

	// Act
	actual := "test"

	// Assert
	assert.Equal(t, expected, actual, "Test framework should work correctly")
}

func TestExampleTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"positive", 5, 5},
		{"zero", 0, 0},
		{"negative", -5, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

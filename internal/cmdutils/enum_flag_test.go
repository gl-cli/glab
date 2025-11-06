//go:build !integration

package cmdutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnumValue(t *testing.T) {
	tests := []struct {
		name       string
		allowed    []string
		defaultVal string
		expected   string
	}{
		{
			name:       "single allowed value",
			allowed:    []string{"option1"},
			defaultVal: "option1",
			expected:   "option1",
		},
		{
			name:       "multiple allowed values",
			allowed:    []string{"option1", "option2", "option3"},
			defaultVal: "option2",
			expected:   "option2",
		},
		{
			name:       "empty allowed values",
			allowed:    []string{},
			defaultVal: "defaultValue",
			expected:   "defaultValue",
		},
		{
			name:       "duplicate allowed values",
			allowed:    []string{"option1", "option1", "option2"},
			defaultVal: "option1",
			expected:   "option1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var value string
			enumValue := NewEnumValue(tt.allowed, tt.defaultVal, &value)

			require.NotNil(t, enumValue, "NewEnumValue should not return nil")
			assert.Equal(t, tt.expected, value, "value should be set to expected default")
			assert.Equal(t, tt.expected, enumValue.String(), "String() should return expected value")

			// Check that allowed values are properly stored
			for _, allowed := range tt.allowed {
				_, ok := enumValue.allowed[allowed]
				assert.True(t, ok, "expected %q to be in allowed values", allowed)
			}
		})
	}
}

func TestNewEnumValue_PanicWithNilPointer(t *testing.T) {
	assert.PanicsWithValue(t, "the given enum flag value cannot be nil", func() {
		NewEnumValue([]string{"option1"}, "default", nil)
	})
}

func TestEnumValue_Set(t *testing.T) {
	tests := []struct {
		name        string
		allowed     []string
		setValue    string
		expectError bool
	}{
		{
			name:        "valid value",
			allowed:     []string{"option1", "option2", "option3"},
			setValue:    "option2",
			expectError: false,
		},
		{
			name:        "invalid value",
			allowed:     []string{"option1", "option2", "option3"},
			setValue:    "invalid",
			expectError: true,
		},
		{
			name:        "empty string value when allowed",
			allowed:     []string{"", "option1", "option2"},
			setValue:    "",
			expectError: false,
		},
		{
			name:        "empty string value when not allowed",
			allowed:     []string{"option1", "option2"},
			setValue:    "",
			expectError: true,
		},
		{
			name:        "case sensitive - different case",
			allowed:     []string{"Option1", "option2"},
			setValue:    "option1",
			expectError: true,
		},
		{
			name:        "case sensitive - exact match",
			allowed:     []string{"Option1", "option2"},
			setValue:    "Option1",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var value string
			enumValue := NewEnumValue(tt.allowed, "default", &value)

			err := enumValue.Set(tt.setValue)

			if tt.expectError {
				assert.Error(t, err, "expected error but got none")
				assert.NotEmpty(t, err.Error(), "expected non-empty error message")
			} else {
				assert.NoError(t, err, "unexpected error")
				assert.Equal(t, tt.setValue, value, "expected value to be set correctly")
				assert.Equal(t, tt.setValue, enumValue.String(), "expected String() to return set value")
			}
		})
	}
}

func TestEnumValue_Set_ErrorMessage(t *testing.T) {
	var value string
	allowed := []string{"option1", "option2", "option3"}
	enumValue := NewEnumValue(allowed, "default", &value)

	err := enumValue.Set("invalid")
	require.Error(t, err, "expected error but got none")

	errorMsg := err.Error()
	expectedPrefix := "must be one of"
	assert.True(t, len(errorMsg) >= len(expectedPrefix), "error message should be long enough")
	assert.Equal(t, expectedPrefix, errorMsg[:len(expectedPrefix)], "error message should start with expected prefix")

	// Check that all allowed values are mentioned in the error
	for _, allowed := range allowed {
		assert.Contains(t, errorMsg, allowed, "expected error message to contain %q", allowed)
	}
}

func TestEnumValue_ValueReference(t *testing.T) {
	var value string
	allowed := []string{"option1", "option2", "option3"}
	enumValue := NewEnumValue(allowed, "option1", &value)

	// Test that the value reference is properly updated
	assert.Equal(t, "option1", value, "expected initial value to be %q", "option1")

	err := enumValue.Set("option2")
	require.NoError(t, err, "unexpected error")

	assert.Equal(t, "option2", value, "expected value reference to be updated to %q", "option2")

	// Test that String() returns the same value as the reference
	assert.Equal(t, value, enumValue.String(), "expected String() to return same as value reference")
}

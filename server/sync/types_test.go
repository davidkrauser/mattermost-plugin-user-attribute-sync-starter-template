package sync

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
)

func TestInferFieldType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected model.PropertyFieldType
	}{
		// Array detection tests
		{
			name:     "interface slice returns multiselect",
			value:    []interface{}{"value1", "value2"},
			expected: model.PropertyFieldTypeMultiselect,
		},
		{
			name:     "string slice returns multiselect",
			value:    []string{"Level1", "Level2"},
			expected: model.PropertyFieldTypeMultiselect,
		},
		{
			name:     "empty interface slice returns multiselect",
			value:    []interface{}{},
			expected: model.PropertyFieldTypeMultiselect,
		},
		{
			name:     "single element array returns multiselect",
			value:    []interface{}{"single"},
			expected: model.PropertyFieldTypeMultiselect,
		},

		// Date detection tests
		{
			name:     "valid date string YYYY-MM-DD returns date",
			value:    "2023-01-15",
			expected: model.PropertyFieldTypeDate,
		},
		{
			name:     "date at start of year returns date",
			value:    "2024-01-01",
			expected: model.PropertyFieldTypeDate,
		},
		{
			name:     "date at end of year returns date",
			value:    "2024-12-31",
			expected: model.PropertyFieldTypeDate,
		},
		{
			name:     "leap year date returns date",
			value:    "2024-02-29",
			expected: model.PropertyFieldTypeDate,
		},

		// Text fallback tests
		{
			name:     "simple string returns text",
			value:    "Engineering",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "empty string returns text",
			value:    "",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "string with spaces returns text",
			value:    "US East",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "numeric string returns text",
			value:    "12345",
			expected: model.PropertyFieldTypeText,
		},

		// Invalid date formats return text
		{
			name:     "date without leading zeros returns text",
			value:    "2023-1-5",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "date with slashes returns text",
			value:    "2023/01/15",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "date with dots returns text",
			value:    "2023.01.15",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "date with time returns text",
			value:    "2023-01-15T10:30:00",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "date with extra characters returns text",
			value:    "2023-01-15 extra",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "partial date returns text",
			value:    "2023-01",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "date with invalid month returns text",
			value:    "2023-13-01",
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "date with invalid day returns text",
			value:    "2023-01-32",
			expected: model.PropertyFieldTypeText,
		},

		// Edge cases
		{
			name:     "nil value returns text",
			value:    nil,
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "integer value returns text",
			value:    42,
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "float value returns text",
			value:    3.14,
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "boolean value returns text",
			value:    true,
			expected: model.PropertyFieldTypeText,
		},
		{
			name:     "map value returns text",
			value:    map[string]interface{}{"key": "value"},
			expected: model.PropertyFieldTypeText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferFieldType(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDatePatternRegex validates the date pattern regex directly
func TestDatePatternRegex(t *testing.T) {
	validDates := []string{
		"2023-01-15",
		"2024-12-31",
		"2020-02-29", // leap year
		"1999-01-01",
		"2099-12-31",
	}

	for _, date := range validDates {
		t.Run("valid:"+date, func(t *testing.T) {
			assert.True(t, datePatternRegex.MatchString(date), "Expected %s to match date pattern", date)
		})
	}

	invalidDates := []string{
		"2023-1-15",        // missing leading zero
		"2023-01-5",        // missing leading zero
		"23-01-15",         // two-digit year
		"2023/01/15",       // wrong separator
		"2023.01.15",       // wrong separator
		"2023-01-15T12:00", // with time
		"2023-01-15 ",      // trailing space
		" 2023-01-15",      // leading space
		"01-15-2023",       // wrong order
		"2023-13-01",       // invalid month
		"2023-01-32",       // invalid day
		"",                 // empty
		"not a date",       // random string
	}

	for _, date := range invalidDates {
		t.Run("invalid:"+date, func(t *testing.T) {
			assert.False(t, datePatternRegex.MatchString(date), "Expected %s to NOT match date pattern", date)
		})
	}
}

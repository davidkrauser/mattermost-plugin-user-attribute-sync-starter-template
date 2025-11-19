package sync

import (
	"encoding/json"
	"fmt"
)

// formatStringValue formats text and date field values for PropertyService.
//
// The PropertyService API requires all PropertyValue.Value fields to be JSON-encoded.
// This function converts a string value (text or date) to json.RawMessage format.
//
// Both text and date fields use the same format - a JSON-encoded string.
//
// Marshaling is required to:
//  1. Add surrounding quotes (JSON strings must be quoted)
//  2. Escape special characters (quotes, backslashes, newlines, etc.)
//
// Args:
//   - value: The string value to format
//
// Returns:
//   - json.RawMessage containing the JSON-encoded string
//   - Error if JSON marshaling fails
//
// Example:
//
//	Input:  "Engineering"
//	Output: json.RawMessage(`"Engineering"`)
//
//	Input:  "2023-01-15"
//	Output: json.RawMessage(`"2023-01-15"`)
func formatStringValue(value string) (json.RawMessage, error) {
	marshaled, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal string value: %w", err)
	}

	return json.RawMessage(marshaled), nil
}

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

// formatMultiselectValue formats multiselect field values for PropertyService.
//
// Multiselect fields store values as arrays of option IDs (not option names).
// This function converts an array of option names from external data into the
// array of option IDs that Mattermost expects.
//
// Uses FieldCache for fast in-memory option name → ID lookups without repeated
// KVStore reads. The cache was populated during field synchronization.
//
// Args:
//   - cache: FieldCache containing option mappings
//   - fieldName: Name of the multiselect field (for cache lookup)
//   - values: Array of option names (e.g., ["Level1", "Level2"])
//
// Returns:
//   - json.RawMessage containing JSON-encoded array of option IDs
//   - Error if any option name not found in cache or marshaling fails
//
// Example:
//
//	Input:  cache (with mappings), fieldName="security_clearance", values=["Level1", "Level2"]
//	Cache:  {"Level1": "opt_abc123", "Level2": "opt_def456"}
//	Output: json.RawMessage(`["opt_abc123","opt_def456"]`)
//
// Missing options are treated as errors because they indicate data inconsistency
// between the external system and Mattermost field definitions.
func formatMultiselectValue(cache FieldCache, fieldName string, values []string) (json.RawMessage, error) {
	// Convert option names to option IDs
	optionIDs := make([]string, 0, len(values))
	for _, optionName := range values {
		optionID, err := cache.GetOptionID(fieldName, optionName)
		if err != nil {
			return nil, fmt.Errorf("failed to get option ID for %s.%s: %w", fieldName, optionName, err)
		}
		if optionID == "" {
			return nil, fmt.Errorf("option %s not found for field %s", optionName, fieldName)
		}
		optionIDs = append(optionIDs, optionID)
	}

	// Marshal array of IDs to JSON
	marshaled, err := json.Marshal(optionIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal multiselect value: %w", err)
	}

	return json.RawMessage(marshaled), nil
}

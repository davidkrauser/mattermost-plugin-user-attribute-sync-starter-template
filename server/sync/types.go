package sync

import (
	"regexp"

	"github.com/mattermost/mattermost/server/public/model"
)

// datePatternRegex matches ISO 8601 date strings in YYYY-MM-DD format.
// This pattern is used to automatically infer date field types from string values.
// It validates:
//   - Year: 4 digits (0000-9999)
//   - Month: 01-12
//   - Day: 01-31
//
// Note: This doesn't validate month-specific day limits (e.g., Feb 30th would match).
var datePatternRegex = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$`)

// inferFieldType determines the appropriate PropertyFieldType for a given value.
// This function enables automatic type inference from JSON data structure without
// requiring manual schema definition.
//
// Type inference rules:
//   - Arrays (slices) → PropertyFieldTypeMultiselect
//   - Strings matching YYYY-MM-DD date pattern → PropertyFieldTypeDate
//   - All other values (including other strings) → PropertyFieldTypeText
//
// Why these rules:
//   - Array detection is straightforward and unambiguous
//   - Date pattern matching provides reliable detection for ISO 8601 dates
//   - Text is a safe fallback that can represent any string value
//
// Limitations:
//   - Type inference happens only at field creation time
//   - Once a field type is set, it cannot be changed (Mattermost CPA constraint)
//   - Only supports 3 of the 6 available Mattermost field types (text, multiselect, date)
//   - Date detection requires strict YYYY-MM-DD format
//
// Parameters:
//   - value: The sample value from which to infer the field type
//
// Returns:
//   - The inferred PropertyFieldType
func inferFieldType(value interface{}) model.PropertyFieldType {
	// Check for nil values - default to text
	if value == nil {
		return model.PropertyFieldTypeText
	}

	// Check if value is an array (slice) - indicates multiselect field
	// This handles any array type, including []interface{} from JSON unmarshaling
	switch v := value.(type) {
	case []interface{}:
		return model.PropertyFieldTypeMultiselect
	case []string:
		return model.PropertyFieldTypeMultiselect
	case string:
		// Check if string matches date pattern (YYYY-MM-DD)
		if datePatternRegex.MatchString(v) {
			return model.PropertyFieldTypeDate
		}
		// Other strings default to text
		return model.PropertyFieldTypeText
	default:
		// All other types default to text
		// This includes numbers, booleans, objects, etc.
		// These would need to be converted to strings when setting values
		return model.PropertyFieldTypeText
	}
}

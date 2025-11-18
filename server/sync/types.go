package sync

import (
	"regexp"
	"strings"
	"unicode"

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

// toDisplayName converts a field name from snake_case or kebab-case to Title Case
// for user-friendly display in the Mattermost UI.
//
// Transformation rules:
//   - Split on underscores (_) and hyphens (-)
//   - Capitalize the first letter of each word
//   - Join words with spaces
//
// Examples:
//   - "security_clearance" → "Security Clearance"
//   - "start-date" → "Start Date"
//   - "department" → "Department"
//   - "user_id" → "User Id"
//
// Why this transformation:
// External systems often use snake_case or kebab-case for field names (e.g., LDAP
// attributes, JSON API fields, database columns). While these naming conventions
// are programmer-friendly, they're not ideal for display in a UI. This function
// provides automatic conversion to human-readable display names.
//
// The internal field name (the key in the JSON data) is preserved and used for
// all lookups. Only the display name shown to users is transformed.
//
// Parameters:
//   - name: The field name to transform (typically from JSON keys)
//
// Returns:
//   - The transformed display name in Title Case with spaces
func toDisplayName(name string) string {
	// Handle empty string
	if name == "" {
		return ""
	}

	// Split on underscores and hyphens
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})

	// Capitalize first letter of each word
	for i, word := range words {
		if word != "" {
			// Convert to runes to properly handle Unicode
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}

	// Join with spaces
	return strings.Join(words, " ")
}

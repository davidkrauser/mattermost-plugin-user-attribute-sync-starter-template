package sync

// discoverFields extracts all unique field names from a collection of user records
// and returns a sample value for each field to enable type inference.
//
// This function enables dynamic field creation without requiring a predefined schema.
// By examining the actual data structure, we can automatically identify which Custom
// Profile Attribute fields need to be created.
//
// Field discovery rules:
//   - Scan all users and collect all field names that appear
//   - Exclude the "email" field (used only for user identification, not as a CPA)
//   - Skip fields with nil values (can't infer type from nil)
//   - For each unique field with a non-nil value, capture a sample value for type inference
//
// Why exclude email:
// The "email" field is used to match external user records to Mattermost users
// via the GetUserByEmail API. It's a lookup key, not a Custom Profile Attribute
// to be synced.
//
// Why skip nil values:
// If a field has a nil value, we can't determine its type. We skip these and rely
// on other user records to provide a non-nil sample value. If ALL users have nil
// for a particular field, that field won't be discovered (which is appropriate since
// we can't create a field without knowing its type).
//
// Why sample values:
// The sample value is passed to inferFieldType() to determine if the field should
// be text, multiselect, or date. We need at least one actual non-nil value to make
// this determination accurately.
//
// Handling varying schemas:
// Different users may have different fields present. For example:
//   - User A: {email, department, location}
//   - User B: {email, department, clearance_level}
//   - Result: {department, location, clearance_level} with sample values
//
// This union approach ensures all fields across all users are discovered and created.
//
// Parameters:
//   - users: Array of user records, each represented as a map of field names to values
//
// Returns:
//   - Map of field name → sample value for all discovered fields (excluding "email")
func discoverFields(users []map[string]interface{}) map[string]interface{} {
	fields := make(map[string]interface{})

	// Scan all users and collect fields
	for _, user := range users {
		for fieldName, value := range user {
			// Skip the email field - it's used for user lookup, not as a CPA field
			if fieldName == "email" {
				continue
			}

			// Skip nil values - we can't infer type from nil
			if value == nil {
				continue
			}

			// If we haven't seen this field yet, add it with this sample value
			if _, exists := fields[fieldName]; !exists {
				fields[fieldName] = value
			}
		}
	}

	return fields
}

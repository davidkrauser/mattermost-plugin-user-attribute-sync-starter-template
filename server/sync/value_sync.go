package sync

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
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
// Uses hardcoded programOptionNameToID mapping for fast lookups without any
// storage or caching overhead.
//
// Args:
//   - fieldName: Name of the multiselect field (must be "programs")
//   - values: Array of option names (e.g., ["Apples", "Oranges"])
//
// Returns:
//   - json.RawMessage containing JSON-encoded array of option IDs
//   - Error if field is not "programs" or any option name not found
//
// Example:
//
//	Input:  fieldName="programs", values=["Apples", "Oranges"]
//	Output: json.RawMessage(`["option_apples","option_oranges"]`)
//
// Missing options are treated as errors because they indicate data inconsistency
// between the external system and the hardcoded schema.
func formatMultiselectValue(fieldName string, values []string) (json.RawMessage, error) {
	// Only "programs" field is multiselect in the hardcoded schema
	if fieldName != "programs" {
		return nil, fmt.Errorf("unexpected multiselect field: %s", fieldName)
	}

	// Convert option names to option IDs using hardcoded mapping
	optionIDs := make([]string, 0, len(values))
	for _, optionName := range values {
		optionID := GetProgramOptionID(optionName)
		if optionID == "" {
			return nil, fmt.Errorf("unknown program option: %s", optionName)
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

// buildPropertyValues constructs PropertyValue objects for all attributes of a user.
//
// This function prepares a batch of PropertyValues for a single user, ready to be
// upserted to Mattermost via the PropertyService API. It uses the hardcoded field
// schema to map external field names to Mattermost field IDs.
//
// The function skips the "email" field (used for user resolution only, not synced as
// an attribute) and continues processing even if individual fields fail, collecting
// errors for reporting.
//
// Args:
//   - api: Mattermost API client for logging
//   - user: The Mattermost user to build values for
//   - groupID: Property group ID (custom_profile_attributes)
//   - userAttrs: Map of attribute names to values from external system
//
// Returns:
//   - Array of PropertyValue objects ready for bulk upsert
//   - Error if critical failure occurs (individual field failures are logged and skipped)
//
// Type handling:
//   - []interface{} or []string → multiselect (convert option names to IDs)
//   - string → text or date field (JSON-encode as string)
//   - Unknown fields are skipped with a warning
func buildPropertyValues(api *pluginapi.Client, user *model.User, groupID string, userAttrs map[string]interface{}) ([]*model.PropertyValue, error) {
	values := make([]*model.PropertyValue, 0, len(userAttrs))

	for fieldName, fieldValue := range userAttrs {
		// Skip email field (used for user resolution only)
		if fieldName == "email" {
			continue
		}

		// Look up field ID from hardcoded mapping
		fieldID := GetFieldID(fieldName)
		if fieldID == "" {
			api.Log.Warn("Unknown field name, skipping",
				"field_name", fieldName,
				"user_email", user.Email)
			continue
		}

		// Format value based on type
		var formattedValue json.RawMessage
		var formatErr error

		switch v := fieldValue.(type) {
		case []interface{}:
			// Multiselect field - convert interface{} array to string array
			stringValues := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					stringValues = append(stringValues, str)
				}
			}
			formattedValue, formatErr = formatMultiselectValue(fieldName, stringValues)

		case []string:
			// Multiselect field - already string array
			formattedValue, formatErr = formatMultiselectValue(fieldName, v)

		case string:
			// Text or date field
			formattedValue, formatErr = formatStringValue(v)

		default:
			api.Log.Warn("Unsupported field value type, skipping field",
				"field_name", fieldName,
				"user_email", user.Email,
				"value_type", fmt.Sprintf("%T", fieldValue))
			continue
		}

		if formatErr != nil {
			api.Log.Warn("Failed to format field value, skipping field",
				"field_name", fieldName,
				"user_email", user.Email,
				"error", formatErr.Error())
			continue
		}

		// Build PropertyValue
		propertyValue := &model.PropertyValue{
			GroupID:    groupID,
			TargetType: "user",
			TargetID:   user.Id,
			FieldID:    fieldID,
			Value:      formattedValue,
		}

		values = append(values, propertyValue)
	}

	return values, nil
}

// SyncUsers synchronizes attribute values for all users from external data.
//
// This is the main orchestrator for value synchronization. It processes each user
// independently, ensuring that failures for individual users don't block the entire
// sync operation. This graceful degradation is critical for production reliability.
//
// For each user:
//  1. Resolve Mattermost user by email
//  2. Build PropertyValues for all attributes using hardcoded field mappings
//  3. Bulk upsert values via PropertyService API
//
// Args:
//   - api: Mattermost API client
//   - groupID: Property group ID (custom_profile_attributes)
//   - users: Array of user attribute maps from external system
//
// Returns:
//   - Error only if critical failure occurs (individual user failures are logged)
//
// Design decisions:
//   - User not found by email → logged as warning, skipped
//   - Empty attributes → skipped (no values to sync)
//   - PropertyValue build failure → logged, skipped
//   - Upsert failure → logged, continue with next user
//
// This partial failure handling ensures progress even when some users have data
// quality issues or have been removed from Mattermost.
//
//nolint:revive // SyncUsers is the conventional name for this orchestrator function
func SyncUsers(api *pluginapi.Client, groupID string, users []map[string]interface{}) error {
	for _, userAttrs := range users {
		// Extract email for user resolution
		email, ok := userAttrs["email"].(string)
		if !ok || email == "" {
			api.Log.Warn("User object missing email field, skipping")
			continue
		}

		// Resolve Mattermost user by email
		user, err := api.User.GetByEmail(email)
		if err != nil {
			api.Log.Warn("User not found by email, skipping",
				"email", email,
				"error", err.Error())
			continue
		}

		// Build PropertyValues for this user
		values, err := buildPropertyValues(api, user, groupID, userAttrs)
		if err != nil {
			api.Log.Error("Failed to build property values, skipping user",
				"user_email", email,
				"error", err.Error())
			continue
		}

		// Skip if no values to sync (e.g., only email field present)
		if len(values) == 0 {
			api.Log.Debug("No property values to sync for user", "email", email)
			continue
		}

		// Bulk upsert all values for this user
		_, err = api.Property.UpsertPropertyValues(values)
		if err != nil {
			api.Log.Error("Failed to upsert property values, skipping user",
				"user_email", email,
				"value_count", len(values),
				"error", err.Error())
			continue
		}

		api.Log.Debug("Successfully synced user attributes",
			"email", email,
			"attribute_count", len(values))
	}

	return nil
}

package sync

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

// createPropertyField creates a new Custom Profile Attribute field in Mattermost.
// This function builds the PropertyField struct with appropriate visibility and management
// attributes, calls the Mattermost API to create the field, and persists the field mapping
// via FieldCache for future lookups.
//
// Field creation process:
//  1. Build PropertyField struct with GroupID, Name, Type, and Attrs
//  2. Set Name using toDisplayName() for user-friendly UI presentation
//  3. Set visibility to "hidden" (not shown in profile or user card by default)
//  4. Set managed to "admin" (users cannot edit, only admins/plugins can set values)
//  5. Call CreatePropertyField API
//  6. Save field name → field ID mapping via FieldCache
//
// Why use FieldCache:
// The field mapping must persist across plugin restarts (FieldCache writes to KVStore)
// and be available in-memory during the current sync (for value synchronization).
// The cache provides both persistence and performance.
//
// Hidden visibility:
// Fields are set to hidden because they're managed by an external system. Users
// shouldn't see these in their profile by default since they're synchronized
// automatically and not meant for manual display/editing in Mattermost.
//
// Admin-managed:
// The "managed": "admin" attribute prevents users from editing these fields through
// the UI. Since values are synchronized from an external source of truth, user edits
// would be overwritten on the next sync. This setting ensures data consistency.
//
// Error handling:
// If field creation fails, the error is returned to the caller. The cache mapping
// is only saved if field creation succeeds. This ensures the plugin can retry creation
// on the next sync if there's a transient failure.
//
// Parameters:
//   - client: pluginapi.Client for accessing Mattermost APIs
//   - groupID: The Custom Profile Attributes group ID
//   - fieldName: The internal field name (e.g., "security_clearance")
//   - fieldType: The field type (text, multiselect, or date)
//   - cache: FieldCache for persisting field mapping
//
// Returns:
//   - The created PropertyField with its assigned ID
//   - Error if field creation or cache save fails
func createPropertyField(
	client *pluginapi.Client,
	groupID string,
	fieldName string,
	fieldType model.PropertyFieldType,
	cache FieldCache,
) (*model.PropertyField, error) {
	// Build the PropertyField struct
	field := &model.PropertyField{
		GroupID: groupID,
		Name:    toDisplayName(fieldName), // Use friendly display name
		Type:    fieldType,
		Attrs: model.StringInterface{
			// Hidden: Don't show in profile/user card (externally managed data)
			model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
			// Admin-managed: Users cannot edit (prevents conflicts with external sync)
			model.CustomProfileAttributesPropertyAttrsManaged: "admin",
		},
	}

	// Create the field via Mattermost API
	createdField, err := client.Property.CreatePropertyField(field)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create property field %s", fieldName)
	}

	// Save the field mapping via FieldCache for future lookups
	// Note: We use the original fieldName (not display name) as the key since that's
	// what appears in the JSON data
	// The cache writes through to KVStore for persistence
	if err := cache.SaveFieldMapping(fieldName, createdField.ID); err != nil {
		// Log error but don't fail - field was created successfully
		// The mapping can be recovered on next sync by querying existing fields
		return createdField, errors.Wrapf(err, "field created but failed to save mapping for %s", fieldName)
	}

	return createdField, nil
}

// extractMultiselectOptions collects all unique option values for a multiselect field
// by examining the field's values across all user records.
//
// This function is used to build the complete set of options that need to exist for
// a multiselect field. It handles the data structure from JSON unmarshaling where
// multiselect values are represented as arrays.
//
// Extraction process:
//  1. Iterate through all user records
//  2. For each user, check if they have the specified field
//  3. If the field value is an array, extract each element as an option name
//  4. Deduplicate options using a set (each unique value appears once)
//  5. Return sorted list for consistency
//
// Why deduplication:
// Multiple users may have the same option values (e.g., multiple users in
// "Engineering" department). We only need to create each unique option once
// in the field definition.
//
// Why extract from all users:
// Unlike field discovery which only needs one sample value, option extraction
// must examine ALL users to find ALL possible option values. For example:
//   - User A: programs: ["Alpha", "Beta"]
//   - User B: programs: ["Beta", "Gamma"]
//   - Result: ["Alpha", "Beta", "Gamma"]
//
// Array value handling:
// JSON unmarshaling produces []interface{} for arrays. Each element needs to
// be type-asserted to string. Non-string elements are skipped since multiselect
// options must be strings.
//
// Parameters:
//   - users: Array of user records to examine
//   - fieldName: The field name to extract options from
//
// Returns:
//   - Slice of unique option names (strings) found across all users
func extractMultiselectOptions(users []map[string]interface{}, fieldName string) []string {
	// Use a set (map with empty struct values) for deduplication
	optionsSet := make(map[string]struct{})

	// Scan all users
	for _, user := range users {
		value, exists := user[fieldName]
		if !exists {
			continue
		}

		// Check if value is an array
		arrayValue, ok := value.([]interface{})
		if !ok {
			// Not an array - skip this user's value
			continue
		}

		// Extract each option from the array
		for _, item := range arrayValue {
			// Options must be strings
			optionName, ok := item.(string)
			if !ok {
				// Skip non-string values
				continue
			}

			// Skip empty strings
			if optionName == "" {
				continue
			}

			// Add to set
			optionsSet[optionName] = struct{}{}
		}
	}

	// Convert set to slice for return
	options := make([]string, 0, len(optionsSet))
	for option := range optionsSet {
		options = append(options, option)
	}

	return options
}

// mergeOptions combines existing multiselect options with new option values, implementing
// the append-only option management strategy.
//
// This function is critical for maintaining data integrity in multiselect fields.
// It ensures that:
//  1. Existing options retain their IDs (prevents orphaning user values)
//  2. New options get new IDs (allows users to select new values)
//  3. No options are ever removed (append-only strategy)
//
// Append-only strategy rationale:
// If an option is removed from a multiselect field, any user values that reference
// that option's ID become orphaned - the UI can't display them, but they still exist
// in the database. This creates data inconsistency and confuses users.
//
// Instead, we append new options while preserving all existing ones. Even if an option
// is no longer present in the source data, we keep it in the field definition so that
// historical user values remain valid.
//
// Merge algorithm:
//  1. Build a map of existing option names → IDs for fast lookup
//  2. Start with all existing options (preserve everything)
//  3. For each new value:
//     - If it matches an existing option name, skip (already in list with correct ID)
//     - If it's new, generate a new ID and append to list
//  4. Return merged list and count of newly added options
//
// ID preservation importance:
// PropertyValue records store option IDs, not option names. If we change an option's
// ID, all existing user values would become invalid. By preserving IDs, we ensure
// that user values continue to work correctly across syncs.
//
// Parameters:
//   - existingOptions: Current options from the field definition (from PropertyService API)
//   - newValues: New option names discovered from current sync data
//
// Returns:
//   - Merged options list (existing + new)
//   - Count of newly added options (0 if all values already existed)
func mergeOptions(existingOptions []map[string]interface{}, newValues []string) ([]interface{}, int) {
	// Build map of existing option names → IDs for fast lookup
	existingMap := make(map[string]string)

	// Start with all existing options (append-only strategy)
	// Use []interface{} which is already registered with gob by Mattermost RPC system
	merged := make([]interface{}, 0, len(existingOptions))

	for _, option := range existingOptions {
		// Extract name and ID from option map
		name, nameOk := option["name"].(string)
		id, idOk := option["id"].(string)
		if nameOk && idOk {
			existingMap[name] = id
			merged = append(merged, map[string]interface{}{
				"id":   id,
				"name": name,
			})
		}
	}

	// Track count of new options added
	newCount := 0

	// Add new options that don't already exist
	for _, value := range newValues {
		// Skip if this option already exists
		if _, exists := existingMap[value]; exists {
			continue
		}

		// Generate new ID for this option
		newID := model.NewId()

		// Create new option
		newOption := map[string]interface{}{
			"id":   newID,
			"name": value,
		}

		// Add to merged list
		merged = append(merged, newOption)
		newCount++

		// Add to map to prevent duplicates within newValues
		existingMap[value] = newID
	}

	return merged, newCount
}

// syncFields orchestrates the complete field synchronization process for Custom Profile Attributes.
// This function coordinates field discovery, type inference, field creation, and option management
// to ensure all fields discovered in the user data exist in Mattermost with correct definitions.
//
// Orchestration flow:
//  1. Discover all unique fields from user data (excluding "email")
//  2. For each discovered field:
//     a. Check if field already exists (lookup via FieldCache)
//     b. If new: infer type, create field, handle multiselect options
//     c. If existing multiselect: extract and merge new options
//  3. Return complete field name → ID mapping for value synchronization
//
// Why orchestrator pattern:
// This function separates high-level coordination logic from low-level operations.
// Each helper function (discoverFields, createPropertyField, mergeOptions) handles
// one specific task with clear inputs/outputs. The orchestrator combines them into
// a coherent workflow, making the code testable and maintainable.
// queryExistingFieldByName searches for an existing property field by name in the given group.
// This is used as a fallback when field creation fails due to the field already existing.
//
// The function uses the SearchPropertyFields API to fetch all fields in the group, then
// searches through them to find one matching the given name and type.
//
// Parameters:
//   - client: pluginapi.Client for accessing Mattermost APIs
//   - groupID: The Custom Profile Attributes group ID
//   - fieldName: The internal field name to search for (e.g., "department")
//   - fieldType: The expected field type (text, multiselect, or date)
//
// Returns:
//   - The found PropertyField, or nil if not found
//   - Error if the API call fails
func queryExistingFieldByName(
	client *pluginapi.Client,
	groupID string,
	fieldName string,
	fieldType model.PropertyFieldType,
) (*model.PropertyField, error) {
	// Search for all fields in the group
	// Using a large PerPage to get all fields in one request (reasonable for typical use cases)
	opts := model.PropertyFieldSearchOpts{
		GroupID:        groupID,
		IncludeDeleted: false,
		PerPage:        1000,
	}

	fields, err := client.Property.SearchPropertyFields(groupID, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to search property fields")
	}

	// Search through fields to find one matching our name
	displayName := toDisplayName(fieldName)
	for _, field := range fields {
		if field.Name == displayName && field.Type == fieldType {
			return field, nil
		}
	}

	// Field not found
	return nil, nil
}

//
// Graceful degradation:
// If a single field fails to create or update, we log the error but continue with
// remaining fields. This prevents one problematic field from blocking the entire
// sync. The field mapping returned will simply exclude the failed field.
//
// FieldCache usage:
// Field IDs are cached via FieldCache to avoid redundant KVStore reads and provide
// in-memory access during value sync. On first sync, fields are created and mappings
// are saved. On subsequent syncs, we use cached mappings (lazy-loaded from KVStore)
// and only update multiselect options if needed.
//
// Multiselect option handling:
//   - New fields: Extract options from data, create field with options
//   - Existing fields: Extract options, query current options, merge, update if changed
//
// Parameters:
//   - client: pluginapi.Client for Mattermost API access
//   - groupID: Custom Profile Attributes group ID
//   - users: User records containing field data
//   - cache: FieldCache for field mapping persistence and in-memory access
//
// Returns:
//   - Map of field name → field ID for all successfully synced fields
//   - Error only if catastrophic failure (returned map may be partial on field-level errors)
//
//nolint:revive // SyncFields is the conventional name for this orchestrator function
func SyncFields(
	client *pluginapi.Client,
	groupID string,
	users []map[string]interface{},
	cache FieldCache,
) (map[string]string, error) {
	// Discover all fields from user data
	discoveredFields := discoverFields(users)

	// Build field mapping (name → ID) as we process each field
	fieldMapping := make(map[string]string)

	// Process each discovered field
	for fieldName, sampleValue := range discoveredFields {
		// Check if field already exists (via cache, lazy-loads from KVStore if needed)
		existingFieldID, err := cache.GetFieldID(fieldName)
		if err != nil {
			// Log error but continue - we'll try to create the field
			client.Log.Warn("Failed to check field mapping from cache",
				"field", fieldName,
				"error", err.Error(),
			)
		}

		if existingFieldID != "" {
			// Field already exists - add to mapping
			fieldMapping[fieldName] = existingFieldID

			// If it's a multiselect field, check if we need to add new options
			fieldType := inferFieldType(sampleValue)
			if fieldType == model.PropertyFieldTypeMultiselect {
				if err = updateMultiselectOptions(client, groupID, existingFieldID, fieldName, users, cache); err != nil {
					client.Log.Warn("Failed to update multiselect options",
						"field", fieldName,
						"error", err.Error(),
					)
					// Continue - field exists and is usable even if option update failed
				}
			}

			continue
		}

		// Field doesn't exist - create it
		fieldType := inferFieldType(sampleValue)

		// For multiselect fields, extract options before creation
		var createdField *model.PropertyField
		if fieldType == model.PropertyFieldTypeMultiselect {
			createdField, err = createMultiselectFieldWithOptions(client, groupID, fieldName, users, cache)
		} else {
			createdField, err = createPropertyField(client, groupID, fieldName, fieldType, cache)
		}

		if err != nil {
			client.Log.Error("Failed to create field",
				"field", fieldName,
				"type", fieldType,
				"error", err.Error(),
			)

			// Fallback: Query API to check if field already exists (duplicate key error case)
			// This handles the scenario where the field exists in DB but not in cache
			existingField, queryErr := queryExistingFieldByName(client, groupID, fieldName, fieldType)
			if queryErr != nil {
				client.Log.Error("Failed to query existing field as fallback",
					"field", fieldName,
					"error", queryErr.Error(),
				)
				// Continue with other fields - graceful degradation
				continue
			}

			if existingField != nil {
				// Field exists! Save to cache and add to mapping
				client.Log.Info("Found existing field via API query",
					"field", fieldName,
					"field_id", existingField.ID,
				)

				// Save to cache for future lookups
				if cacheErr := cache.SaveFieldMapping(fieldName, existingField.ID); cacheErr != nil {
					client.Log.Warn("Failed to save field mapping to cache",
						"field", fieldName,
						"error", cacheErr.Error(),
					)
					// Don't fail - we can still use the field
				}

				// Add to mapping for this sync
				fieldMapping[fieldName] = existingField.ID

				// If it's a multiselect field, check if we need to add new options
				if fieldType == model.PropertyFieldTypeMultiselect {
					if updateErr := updateMultiselectOptions(client, groupID, existingField.ID, fieldName, users, cache); updateErr != nil {
						client.Log.Warn("Failed to update multiselect options for recovered field",
							"field", fieldName,
							"error", updateErr.Error(),
						)
						// Continue - field exists and is usable even if option update failed
					}
				}

				continue
			}

			// Field truly doesn't exist and we couldn't create it - graceful degradation
			client.Log.Error("Field does not exist and could not be created",
				"field", fieldName,
			)
			continue
		}

		// Add to mapping
		fieldMapping[fieldName] = createdField.ID
	}

	return fieldMapping, nil
}

// createMultiselectFieldWithOptions creates a multiselect field with initial options extracted
// from user data.
func createMultiselectFieldWithOptions(
	client *pluginapi.Client,
	groupID string,
	fieldName string,
	users []map[string]interface{},
	cache FieldCache,
) (*model.PropertyField, error) {
	// Extract all unique option values from user data
	optionValues := extractMultiselectOptions(users, fieldName)

	// Build options list with generated IDs
	// Use []interface{} which is already registered with gob by Mattermost RPC system
	options := make([]interface{}, len(optionValues))
	for i, value := range optionValues {
		options[i] = map[string]interface{}{
			"id":   model.NewId(),
			"name": value,
		}
	}

	// Create field with options in Attrs
	field := &model.PropertyField{
		GroupID: groupID,
		Name:    toDisplayName(fieldName),
		Type:    model.PropertyFieldTypeMultiselect,
		Attrs: model.StringInterface{
			model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
			model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
			model.PropertyFieldAttributeOptions:                  options,
		},
	}

	createdField, err := client.Property.CreatePropertyField(field)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create multiselect field %s", fieldName)
	}

	// Save field mapping via cache
	if err := cache.SaveFieldMapping(fieldName, createdField.ID); err != nil {
		return createdField, errors.Wrapf(err, "field created but failed to save mapping for %s", fieldName)
	}

	// Save initial options via cache for future merging
	optionsMap := make(map[string]string)
	for _, opt := range options {
		if optMap, ok := opt.(map[string]interface{}); ok {
			if name, nameOk := optMap["name"].(string); nameOk {
				if id, idOk := optMap["id"].(string); idOk {
					optionsMap[name] = id
				}
			}
		}
	}
	if err := cache.SaveFieldOptions(fieldName, optionsMap); err != nil {
		client.Log.Warn("Failed to save field options via cache",
			"field", fieldName,
			"error", err.Error(),
		)
	}

	return createdField, nil
}

// updateMultiselectOptions checks if new options need to be added to an existing multiselect field
// and updates the field if necessary.
func updateMultiselectOptions(
	client *pluginapi.Client,
	groupID string,
	fieldID string,
	fieldName string,
	users []map[string]interface{},
	cache FieldCache,
) error {
	// Extract current option values from user data
	newOptionValues := extractMultiselectOptions(users, fieldName)
	if len(newOptionValues) == 0 {
		return nil // No options to add
	}

	// Get current field definition
	field, err := client.Property.GetPropertyField(groupID, fieldID)
	if err != nil {
		return errors.Wrapf(err, "failed to get field %s", fieldName)
	}

	// Extract existing options from field
	existingOptions := []map[string]interface{}{}
	if optionsAttr, ok := field.Attrs[model.PropertyFieldAttributeOptions]; ok {
		if optionsArray, ok := optionsAttr.([]interface{}); ok {
			for _, opt := range optionsArray {
				if optMap, ok := opt.(map[string]interface{}); ok {
					existingOptions = append(existingOptions, optMap)
				}
			}
		}
	}

	// Merge options (append-only)
	mergedOptions, newCount := mergeOptions(existingOptions, newOptionValues)

	// If no new options, nothing to update
	if newCount == 0 {
		return nil
	}

	// Update field with merged options
	field.Attrs[model.PropertyFieldAttributeOptions] = mergedOptions
	_, err = client.Property.UpdatePropertyField(groupID, field)
	if err != nil {
		return errors.Wrapf(err, "failed to update options for field %s", fieldName)
	}

	// Update cache with new options mapping
	optionsMap := make(map[string]string)
	for _, opt := range mergedOptions {
		if optMap, ok := opt.(map[string]interface{}); ok {
			if name, nameOk := optMap["name"].(string); nameOk {
				if id, idOk := optMap["id"].(string); idOk {
					optionsMap[name] = id
				}
			}
		}
	}
	if err := cache.SaveFieldOptions(fieldName, optionsMap); err != nil {
		client.Log.Warn("Failed to save updated field options via cache",
			"field", fieldName,
			"error", err.Error(),
		)
	}

	client.Log.Info("Added new options to multiselect field",
		"field", fieldName,
		"new_count", newCount,
	)

	return nil
}

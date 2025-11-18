package sync

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"

	"github.com/mattermost/user-attribute-sync-starter-template/server/store/kvstore"
)

// createPropertyField creates a new Custom Profile Attribute field in Mattermost.
// This function builds the PropertyField struct with appropriate visibility and management
// attributes, calls the Mattermost API to create the field, and persists the field mapping
// in KVStore for future lookups.
//
// Field creation process:
//  1. Build PropertyField struct with GroupID, Name, Type, and Attrs
//  2. Set Name using toDisplayName() for user-friendly UI presentation
//  3. Set visibility to "hidden" (not shown in profile or user card by default)
//  4. Set managed to "admin" (users cannot edit, only admins/plugins can set values)
//  5. Call CreatePropertyField API
//  6. Save field name → field ID mapping in KVStore
//
// Why save to KVStore:
// The field mapping must persist across plugin restarts. On subsequent syncs, we
// check KVStore first to avoid creating duplicate fields. Field IDs are needed for
// all value synchronization operations.
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
// If field creation fails, the error is returned to the caller. The KVStore mapping
// is only saved if field creation succeeds. This ensures the plugin can retry creation
// on the next sync if there's a transient failure.
//
// Parameters:
//   - client: pluginapi.Client for accessing Mattermost APIs
//   - groupID: The Custom Profile Attributes group ID
//   - fieldName: The internal field name (e.g., "security_clearance")
//   - fieldType: The field type (text, multiselect, or date)
//   - store: KVStore interface for persisting field mapping
//
// Returns:
//   - The created PropertyField with its assigned ID
//   - Error if field creation or KVStore save fails
func createPropertyField(
	client *pluginapi.Client,
	groupID string,
	fieldName string,
	fieldType model.PropertyFieldType,
	store kvstore.KVStore,
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

	// Save the field mapping to KVStore for future lookups
	// Note: We use the original fieldName (not display name) as the key since that's
	// what appears in the JSON data
	if err := store.SaveFieldMapping(fieldName, createdField.ID); err != nil {
		// Log error but don't fail - field was created successfully
		// The mapping can be recovered on next sync by querying existing fields
		return createdField, errors.Wrapf(err, "field created but failed to save mapping for %s", fieldName)
	}

	return createdField, nil
}

package sync

import (
	"encoding/json"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

// FieldIDCache stores mappings from external field/option names to Mattermost-generated IDs.
// These IDs are dynamically loaded during plugin activation by creating fields and looking up their IDs.
type FieldIDCache struct {
	// Maps external field names (e.g., "job_title") to Mattermost field IDs
	FieldNameToID map[string]string
	// Maps external option names (e.g., "Apples") to Mattermost option IDs for the "programs" field
	ProgramOptionNameToID map[string]string
}

// GetFieldID translates an external field name to its Mattermost field ID.
func (c *FieldIDCache) GetFieldID(fieldName string) string {
	return c.FieldNameToID[fieldName]
}

// GetProgramOptionID translates an external multiselect option name to its Mattermost option ID.
func (c *FieldIDCache) GetProgramOptionID(optionName string) string {
	return c.ProgramOptionNameToID[optionName]
}

// fieldDefinition defines a Custom Profile Attribute field schema.
type fieldDefinition struct {
	Name           string                   // Display name shown in UI
	ExternalName   string                   // Name used in external data source
	Type           model.PropertyFieldType  // Field type (text, date, multiselect, etc.)
	OptionNames    []string                 // Option names for multiselect fields
}

// fieldDefinitions contains all Custom Profile Attribute fields this plugin creates.
// Custom Profile Attributes (CPAs) are user metadata fields that appear in user profiles.
// This plugin ensures these fields exist on startup and syncs external data into them.
var fieldDefinitions = []fieldDefinition{
	{
		Name:         "Job Title",
		ExternalName: "job_title",
		Type:         model.PropertyFieldTypeText,
	},
	{
		Name:         "Programs",
		ExternalName: "programs",
		Type:         model.PropertyFieldTypeMultiselect,
		OptionNames:  []string{"Apples", "Oranges", "Lemons"},
	},
	{
		Name:         "Start Date",
		ExternalName: "start_date",
		Type:         model.PropertyFieldTypeDate,
	},
}

// createOrUpdateField ensures a CPA field exists and matches the definition.
// Returns the field ID (either existing or newly created).
func createOrUpdateField(
	client *pluginapi.Client,
	groupID string,
	def fieldDefinition,
) (string, error) {
	// Check if field already exists by looking it up by name
	existingField, err := client.Property.GetPropertyFieldByName(groupID, "", def.Name)

	if err == nil && existingField != nil {
		// Field exists - update it to match our definition
		client.Log.Info("Field exists, updating to match definition",
			"field_id", existingField.ID,
			"name", def.Name)

		existingField.Type = def.Type
		existingField.Attrs[model.CustomProfileAttributesPropertyAttrsVisibility] = model.CustomProfileAttributesVisibilityHidden
		existingField.Attrs[model.CustomProfileAttributesPropertyAttrsManaged] = "admin"

		if def.Type == model.PropertyFieldTypeMultiselect {
			// Build options array with name only - Mattermost will generate IDs
			options := make([]interface{}, len(def.OptionNames))
			for i, optionName := range def.OptionNames {
				options[i] = map[string]interface{}{
					"name": optionName,
				}
			}
			existingField.Attrs[model.PropertyFieldAttributeOptions] = options
		}

		_, updateErr := client.Property.UpdatePropertyField(groupID, existingField)
		if updateErr != nil {
			return "", errors.Wrapf(updateErr, "failed to update existing field %s", def.Name)
		}

		client.Log.Info("Updated field successfully", "field_id", existingField.ID, "name", def.Name)
		return existingField.ID, nil
	}

	// Field doesn't exist - create it
	client.Log.Info("Field does not exist, creating", "name", def.Name)

	field := &model.PropertyField{
		// ID left empty - Mattermost will auto-generate
		GroupID: groupID,
		Name:    def.Name,
		Type:    def.Type,
		Attrs: model.StringInterface{
			// Hidden from UI because data is managed externally
			model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
			// Admin-managed prevents users from editing and conflicting with sync
			model.CustomProfileAttributesPropertyAttrsManaged: "admin",
		},
	}

	// Multiselect fields need their options defined
	if def.Type == model.PropertyFieldTypeMultiselect {
		// Build options array with name only - Mattermost will generate IDs
		options := make([]interface{}, len(def.OptionNames))
		for i, optionName := range def.OptionNames {
			options[i] = map[string]interface{}{
				"name": optionName,
			}
		}
		field.Attrs[model.PropertyFieldAttributeOptions] = options
	}

	createdField, createErr := client.Property.CreatePropertyField(field)
	if createErr != nil {
		return "", errors.Wrapf(createErr, "failed to create field %s", def.Name)
	}

	client.Log.Info("Created field successfully", "field_id", createdField.ID, "name", def.Name)
	return createdField.ID, nil
}

// SyncFields ensures all CPA fields exist and match the definitions.
// Returns a FieldIDCache containing mappings from external names to Mattermost-generated IDs.
//
//nolint:revive // SyncFields is the conventional name for this orchestrator function
func SyncFields(client *pluginapi.Client, groupID string) (*FieldIDCache, error) {
	client.Log.Info("Syncing field definitions", "field_count", len(fieldDefinitions))

	cache := &FieldIDCache{
		FieldNameToID:         make(map[string]string),
		ProgramOptionNameToID: make(map[string]string),
	}

	var failedFields []string

	for _, def := range fieldDefinitions {
		fieldID, err := createOrUpdateField(client, groupID, def)
		if err != nil {
			client.Log.Error("Failed to create or update field",
				"name", def.Name,
				"error", err.Error())
			failedFields = append(failedFields, def.Name)
			// Continue with next field for graceful degradation
			continue
		}

		// Store the field name to ID mapping
		cache.FieldNameToID[def.ExternalName] = fieldID

		// For multiselect fields, extract option IDs
		if def.Type == model.PropertyFieldTypeMultiselect && len(def.OptionNames) > 0 {
			// Look up the field again to get the option IDs that Mattermost generated
			field, err := client.Property.GetPropertyField(groupID, fieldID)
			if err != nil {
				client.Log.Error("Failed to get field for option extraction",
					"name", def.Name,
					"field_id", fieldID,
					"error", err.Error())
				continue
			}

			// Extract option IDs from the field attributes
			if optionsRaw, ok := field.Attrs[model.PropertyFieldAttributeOptions]; ok {
				// Convert options to JSON and back to extract IDs
				optionsJSON, err := json.Marshal(optionsRaw)
				if err != nil {
					client.Log.Error("Failed to marshal options",
						"name", def.Name,
						"error", err.Error())
					continue
				}

				var options []map[string]interface{}
				if err := json.Unmarshal(optionsJSON, &options); err != nil {
					client.Log.Error("Failed to unmarshal options",
						"name", def.Name,
						"error", err.Error())
					continue
				}

				// Build option name to ID mapping (only for "programs" field currently)
				if def.ExternalName == "programs" {
					for _, opt := range options {
						if name, ok := opt["name"].(string); ok {
							if id, ok := opt["id"].(string); ok {
								cache.ProgramOptionNameToID[name] = id
							}
						}
					}
					client.Log.Debug("Extracted program option IDs",
						"option_count", len(cache.ProgramOptionNameToID))
				}
			}
		}
	}

	if len(failedFields) > 0 {
		client.Log.Warn("Some fields failed to sync",
			"failed_count", len(failedFields),
			"failed_fields", failedFields)
		// Return partial cache even on failures
	}

	client.Log.Info("Field sync completed",
		"total", len(fieldDefinitions),
		"failed", len(failedFields),
		"fields_cached", len(cache.FieldNameToID),
		"options_cached", len(cache.ProgramOptionNameToID))

	return cache, nil
}

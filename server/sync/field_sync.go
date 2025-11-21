package sync

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

// Field and option ID constants - these are used consistently across the plugin
// to reference specific Custom Profile Attribute fields and their options.
const (
	FieldIDJobTitle  = "field_job_title"
	FieldIDPrograms  = "field_programs"
	FieldIDStartDate = "field_start_date"

	// Program multiselect options
	OptionIDApples  = "option_apples"
	OptionIDOranges = "option_oranges"
	OptionIDLemons  = "option_lemons"
)

// fieldNameToID maps JSON field names from external data to field IDs.
// This is the translation layer between external system field names and
// Mattermost Custom Profile Attribute field IDs.
var fieldNameToID = map[string]string{
	"job_title":  FieldIDJobTitle,
	"programs":   FieldIDPrograms,
	"start_date": FieldIDStartDate,
}

// programOptionNameToID maps program option names to option IDs.
// Used during value synchronization to convert external option names
// to Mattermost option IDs.
var programOptionNameToID = map[string]string{
	"Apples":  OptionIDApples,
	"Oranges": OptionIDOranges,
	"Lemons":  OptionIDLemons,
}

// fieldDefinition represents a hardcoded field schema that should exist
// in Mattermost's Custom Profile Attributes.
type fieldDefinition struct {
	ID      string
	Name    string
	Type    model.PropertyFieldType
	Options []map[string]interface{} // Only for multiselect fields
}

// fieldDefinitions is the hardcoded schema for all Custom Profile Attribute
// fields that this plugin manages. When the plugin starts, it ensures all
// these fields exist in Mattermost with the exact IDs and definitions specified.
//
// Why hardcoded schema:
// - Simple and explicit - developers can see exactly what fields are created
// - No type inference complexity - field types are clearly defined
// - Predictable behavior - no surprises from data structure changes
// - Easy to customize - developers modify this array to match their needs
//
// Each field definition includes:
// - ID: Unique identifier for the field (human-readable)
// - Name: Display name shown in the Mattermost UI
// - Type: Field type (text, date, multiselect)
// - Options: For multiselect fields, the list of available options
var fieldDefinitions = []fieldDefinition{
	{
		ID:   FieldIDJobTitle,
		Name: "Job Title",
		Type: model.PropertyFieldTypeText,
	},
	{
		ID:   FieldIDPrograms,
		Name: "Programs",
		Type: model.PropertyFieldTypeMultiselect,
		Options: []map[string]interface{}{
			{"id": OptionIDApples, "name": "Apples"},
			{"id": OptionIDOranges, "name": "Oranges"},
			{"id": OptionIDLemons, "name": "Lemons"},
		},
	},
	{
		ID:   FieldIDStartDate,
		Name: "Start Date",
		Type: model.PropertyFieldTypeDate,
	},
}

// GetFieldID returns the Mattermost field ID for a given external field name.
// Returns empty string if the field name is not recognized.
//
// This is the primary lookup function used during value synchronization to
// translate external field names to Mattermost field IDs.
func GetFieldID(fieldName string) string {
	return fieldNameToID[fieldName]
}

// GetProgramOptionID returns the Mattermost option ID for a given program name.
// Returns empty string if the option name is not recognized.
//
// This is used during value synchronization to translate external multiselect
// option names to Mattermost option IDs.
func GetProgramOptionID(optionName string) string {
	return programOptionNameToID[optionName]
}

// createOrUpdateField creates or updates a single Custom Profile Attribute field
// based on the hardcoded definition. This function is idempotent - it can be called
// multiple times safely.
//
// The function attempts to create the field. If creation fails because the field
// already exists, it retrieves the existing field and updates it if necessary to
// match the desired definition.
//
// For multiselect fields, options are always set to match the hardcoded definition.
// This ensures the field definition remains consistent with the plugin's expectations.
//
// Parameters:
//   - client: pluginapi.Client for accessing Mattermost APIs
//   - groupID: The Custom Profile Attributes group ID
//   - def: The field definition to create or update
//
// Returns error if field cannot be created or updated.
func createOrUpdateField(
	client *pluginapi.Client,
	groupID string,
	def fieldDefinition,
) error {
	// Build the PropertyField struct
	field := &model.PropertyField{
		ID:      def.ID,
		GroupID: groupID,
		Name:    def.Name,
		Type:    def.Type,
		Attrs: model.StringInterface{
			// Hidden: Don't show in profile/user card (externally managed data)
			model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
			// Admin-managed: Users cannot edit (prevents conflicts with external sync)
			model.CustomProfileAttributesPropertyAttrsManaged: "admin",
		},
	}

	// Add options for multiselect fields
	if def.Type == model.PropertyFieldTypeMultiselect {
		// Convert to []interface{} which is required by the API
		options := make([]interface{}, len(def.Options))
		for i, opt := range def.Options {
			options[i] = opt
		}
		field.Attrs[model.PropertyFieldAttributeOptions] = options
	}

	// Attempt to create the field
	_, err := client.Property.CreatePropertyField(field)
	if err == nil {
		client.Log.Info("Created field successfully", "field_id", def.ID, "name", def.Name)
		return nil
	}

	// If creation failed, check if it's because the field already exists
	client.Log.Debug("Field creation returned error, checking if field exists",
		"field_id", def.ID,
		"error", err.Error())

	// Try to get the existing field
	existingField, getErr := client.Property.GetPropertyField(groupID, def.ID)
	if getErr != nil {
		// Field doesn't exist and we couldn't create it - this is an error
		return errors.Wrapf(err, "failed to create field %s and it doesn't exist", def.ID)
	}

	if existingField == nil {
		return errors.Wrapf(err, "failed to create field %s and retrieval returned nil", def.ID)
	}

	// Field exists - update it to ensure it matches our definition
	client.Log.Info("Field already exists, updating to match definition",
		"field_id", def.ID,
		"name", def.Name)

	// Update the field attributes to match our definition
	existingField.Name = def.Name
	existingField.Type = def.Type
	existingField.Attrs[model.CustomProfileAttributesPropertyAttrsVisibility] = model.CustomProfileAttributesVisibilityHidden
	existingField.Attrs[model.CustomProfileAttributesPropertyAttrsManaged] = "admin"

	// Update options for multiselect fields
	if def.Type == model.PropertyFieldTypeMultiselect {
		options := make([]interface{}, len(def.Options))
		for i, opt := range def.Options {
			options[i] = opt
		}
		existingField.Attrs[model.PropertyFieldAttributeOptions] = options
	}

	_, updateErr := client.Property.UpdatePropertyField(groupID, existingField)
	if updateErr != nil {
		return errors.Wrapf(updateErr, "failed to update existing field %s", def.ID)
	}

	client.Log.Info("Updated field successfully", "field_id", def.ID, "name", def.Name)
	return nil
}

// SyncFields ensures all hardcoded field definitions exist in Mattermost.
// This function should be called during plugin initialization or at the start
// of each sync operation to ensure the field schema is properly set up.
//
// The function iterates through all hardcoded field definitions and creates
// or updates each field. If a field already exists, it's updated to match
// the hardcoded definition.
//
// Graceful degradation:
// If a single field fails to create or update, the error is logged but the
// function continues processing remaining fields. This prevents one problematic
// field from blocking the entire sync.
//
// Parameters:
//   - client: pluginapi.Client for Mattermost API access
//   - groupID: Custom Profile Attributes group ID
//
// Returns error only if critical failure occurs (individual field failures
// are logged but don't cause function failure).
//
//nolint:revive // SyncFields is the conventional name for this orchestrator function
func SyncFields(client *pluginapi.Client, groupID string) error {
	client.Log.Info("Syncing hardcoded field definitions", "field_count", len(fieldDefinitions))

	// Track if any fields failed
	var failedFields []string

	// Create or update each field
	for _, def := range fieldDefinitions {
		if err := createOrUpdateField(client, groupID, def); err != nil {
			client.Log.Error("Failed to create or update field",
				"field_id", def.ID,
				"name", def.Name,
				"error", err.Error())
			failedFields = append(failedFields, def.ID)
			// Continue with next field - graceful degradation
		}
	}

	if len(failedFields) > 0 {
		client.Log.Warn("Some fields failed to sync",
			"failed_count", len(failedFields),
			"failed_fields", failedFields)
		// Don't return error - partial success is acceptable
	}

	client.Log.Info("Field sync completed", "total", len(fieldDefinitions), "failed", len(failedFields))
	return nil
}

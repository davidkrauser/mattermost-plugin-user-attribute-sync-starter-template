package sync

import (
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

// Field and option IDs that this plugin manages in Mattermost's Custom Profile Attributes system.
const (
	FieldIDJobTitle  = "field_job_title"
	FieldIDPrograms  = "field_programs"
	FieldIDStartDate = "field_start_date"

	OptionIDApples  = "option_apples"
	OptionIDOranges = "option_oranges"
	OptionIDLemons  = "option_lemons"
)

// fieldNameToID translates external system field names to Mattermost field IDs.
var fieldNameToID = map[string]string{
	"job_title":  FieldIDJobTitle,
	"programs":   FieldIDPrograms,
	"start_date": FieldIDStartDate,
}

// programOptionNameToID translates external multiselect option names to Mattermost option IDs.
var programOptionNameToID = map[string]string{
	"Apples":  OptionIDApples,
	"Oranges": OptionIDOranges,
	"Lemons":  OptionIDLemons,
}

// fieldDefinition defines a Custom Profile Attribute field schema.
type fieldDefinition struct {
	ID      string
	Name    string
	Type    model.PropertyFieldType
	Options []map[string]interface{} // For multiselect fields only
}

// fieldDefinitions contains all Custom Profile Attribute fields this plugin creates.
// Custom Profile Attributes (CPAs) are user metadata fields that appear in user profiles.
// This plugin ensures these fields exist on startup and syncs external data into them.
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

// GetFieldID translates an external field name to its Mattermost field ID.
func GetFieldID(fieldName string) string {
	return fieldNameToID[fieldName]
}

// GetProgramOptionID translates an external multiselect option name to its Mattermost option ID.
func GetProgramOptionID(optionName string) string {
	return programOptionNameToID[optionName]
}

// createOrUpdateField ensures a CPA field exists and matches the hardcoded definition.
func createOrUpdateField(
	client *pluginapi.Client,
	groupID string,
	def fieldDefinition,
) error {
	// Check if field already exists
	existingField, err := client.Property.GetPropertyField(groupID, def.ID)

	if err == nil && existingField != nil {
		// Field exists - update it to match our definition
		client.Log.Info("Field exists, updating to match definition",
			"field_id", def.ID,
			"name", def.Name)

		existingField.Name = def.Name
		existingField.Type = def.Type
		existingField.Attrs[model.CustomProfileAttributesPropertyAttrsVisibility] = model.CustomProfileAttributesVisibilityHidden
		existingField.Attrs[model.CustomProfileAttributesPropertyAttrsManaged] = "admin"

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

	// Field doesn't exist - create it
	client.Log.Info("Field does not exist, creating",
		"field_id", def.ID,
		"name", def.Name)

	field := &model.PropertyField{
		ID:      def.ID,
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
		options := make([]interface{}, len(def.Options))
		for i, opt := range def.Options {
			options[i] = opt
		}
		field.Attrs[model.PropertyFieldAttributeOptions] = options
	}

	_, createErr := client.Property.CreatePropertyField(field)
	if createErr != nil {
		return errors.Wrapf(createErr, "failed to create field %s", def.ID)
	}

	client.Log.Info("Created field successfully", "field_id", def.ID, "name", def.Name)
	return nil
}

// SyncFields ensures all CPA fields exist and match the hardcoded definitions.
//
//nolint:revive // SyncFields is the conventional name for this orchestrator function
func SyncFields(client *pluginapi.Client, groupID string) error {
	client.Log.Info("Syncing hardcoded field definitions", "field_count", len(fieldDefinitions))

	var failedFields []string

	for _, def := range fieldDefinitions {
		if err := createOrUpdateField(client, groupID, def); err != nil {
			client.Log.Error("Failed to create or update field",
				"field_id", def.ID,
				"name", def.Name,
				"error", err.Error())
			failedFields = append(failedFields, def.ID)
			// Continue with next field for graceful degradation
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

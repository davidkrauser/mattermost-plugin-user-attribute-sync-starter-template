package sync

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockFieldCache is a mock implementation of the FieldCache interface for testing
type mockFieldCache struct {
	mock.Mock
}

func (m *mockFieldCache) GetFieldID(fieldName string) (string, error) {
	args := m.Called(fieldName)
	return args.String(0), args.Error(1)
}

func (m *mockFieldCache) GetOptionID(fieldName, optionName string) (string, error) {
	args := m.Called(fieldName, optionName)
	return args.String(0), args.Error(1)
}

func (m *mockFieldCache) SaveFieldMapping(fieldName, fieldID string) error {
	args := m.Called(fieldName, fieldID)
	return args.Error(0)
}

func (m *mockFieldCache) SaveFieldOptions(fieldName string, options map[string]string) error {
	args := m.Called(fieldName, options)
	return args.Error(0)
}

func TestCreatePropertyField(t *testing.T) {
	groupID := "test-group-id"

	t.Run("successfully creates text field", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		// Expected field structure
		expectedField := &model.PropertyField{
			GroupID: groupID,
			Name:    "Department", // Display name
			Type:    model.PropertyFieldTypeText,
			Attrs: model.StringInterface{
				model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
				model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
			},
		}

		// Returned field (with ID assigned by API)
		createdField := &model.PropertyField{
			ID:      "field-id-123",
			GroupID: groupID,
			Name:    "Department",
			Type:    model.PropertyFieldTypeText,
			Attrs: model.StringInterface{
				model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
				model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
			},
		}

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.GroupID == expectedField.GroupID &&
				f.Name == expectedField.Name &&
				f.Type == expectedField.Type &&
				f.Attrs[model.CustomProfileAttributesPropertyAttrsVisibility] == model.CustomProfileAttributesVisibilityHidden &&
				f.Attrs[model.CustomProfileAttributesPropertyAttrsManaged] == "admin"
		})).Return(createdField, nil)

		cache.On("SaveFieldMapping", "department", "field-id-123").Return(nil)

		result, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, cache)

		require.NoError(t, err)
		assert.Equal(t, "field-id-123", result.ID)
		assert.Equal(t, "Department", result.Name)
		assert.Equal(t, model.PropertyFieldTypeText, result.Type)
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("successfully creates multiselect field", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		createdField := &model.PropertyField{
			ID:      "field-id-456",
			GroupID: groupID,
			Name:    "Security Clearance",
			Type:    model.PropertyFieldTypeMultiselect,
			Attrs: model.StringInterface{
				model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
				model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
			},
		}

		api.On("CreatePropertyField", mock.Anything).Return(createdField, nil)
		cache.On("SaveFieldMapping", "security_clearance", "field-id-456").Return(nil)

		result, err := createPropertyField(client, groupID, "security_clearance", model.PropertyFieldTypeMultiselect, cache)

		require.NoError(t, err)
		assert.Equal(t, "field-id-456", result.ID)
		assert.Equal(t, model.PropertyFieldTypeMultiselect, result.Type)
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("successfully creates date field", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		createdField := &model.PropertyField{
			ID:      "field-id-789",
			GroupID: groupID,
			Name:    "Start Date",
			Type:    model.PropertyFieldTypeDate,
			Attrs: model.StringInterface{
				model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
				model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
			},
		}

		api.On("CreatePropertyField", mock.Anything).Return(createdField, nil)
		cache.On("SaveFieldMapping", "start_date", "field-id-789").Return(nil)

		result, err := createPropertyField(client, groupID, "start_date", model.PropertyFieldTypeDate, cache)

		require.NoError(t, err)
		assert.Equal(t, "field-id-789", result.ID)
		assert.Equal(t, model.PropertyFieldTypeDate, result.Type)
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("uses display name transformation", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		createdField := &model.PropertyField{
			ID:      "field-id",
			GroupID: groupID,
			Name:    "User Access Level", // Transformed from user_access_level
			Type:    model.PropertyFieldTypeText,
			Attrs: model.StringInterface{
				model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
				model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
			},
		}

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.Name == "User Access Level"
		})).Return(createdField, nil)

		cache.On("SaveFieldMapping", "user_access_level", "field-id").Return(nil)

		result, err := createPropertyField(client, groupID, "user_access_level", model.PropertyFieldTypeText, cache)

		require.NoError(t, err)
		assert.Equal(t, "User Access Level", result.Name)
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("sets correct visibility attribute", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		createdField := &model.PropertyField{
			ID:      "field-id",
			GroupID: groupID,
			Name:    "Department",
			Type:    model.PropertyFieldTypeText,
			Attrs: model.StringInterface{
				model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
				model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
			},
		}

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			visibility, ok := f.Attrs[model.CustomProfileAttributesPropertyAttrsVisibility]
			return ok && visibility == model.CustomProfileAttributesVisibilityHidden
		})).Return(createdField, nil)

		cache.On("SaveFieldMapping", "department", "field-id").Return(nil)

		_, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, cache)

		require.NoError(t, err)
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("sets managed attribute to admin", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		createdField := &model.PropertyField{
			ID:      "field-id",
			GroupID: groupID,
			Name:    "Department",
			Type:    model.PropertyFieldTypeText,
			Attrs: model.StringInterface{
				model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
				model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
			},
		}

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			managed, ok := f.Attrs[model.CustomProfileAttributesPropertyAttrsManaged]
			return ok && managed == "admin"
		})).Return(createdField, nil)

		cache.On("SaveFieldMapping", "department", "field-id").Return(nil)

		_, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, cache)

		require.NoError(t, err)
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("returns error when API call fails", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		apiError := errors.New("API error: insufficient permissions")
		api.On("CreatePropertyField", mock.Anything).Return(nil, apiError)

		result, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, cache)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create property field")
		api.AssertExpectations(t)
		// KVStore should not be called if API fails
		cache.AssertNotCalled(t, "SaveFieldMapping")
	})

	t.Run("returns field but logs error when KVStore save fails", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		createdField := &model.PropertyField{
			ID:      "field-id-123",
			GroupID: groupID,
			Name:    "Department",
			Type:    model.PropertyFieldTypeText,
			Attrs: model.StringInterface{
				model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityHidden,
				model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
			},
		}

		api.On("CreatePropertyField", mock.Anything).Return(createdField, nil)
		kvError := errors.New("KVStore error: connection timeout")
		cache.On("SaveFieldMapping", "department", "field-id-123").Return(kvError)

		result, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, cache)

		// Field should still be returned even though KVStore save failed
		require.Error(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "field-id-123", result.ID)
		assert.Contains(t, err.Error(), "failed to save mapping")
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})
}

func TestExtractMultiselectOptions(t *testing.T) {
	t.Run("extracts options from single user", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":    "user@example.com",
				"programs": []interface{}{"Alpha", "Beta"},
			},
		}

		options := extractMultiselectOptions(users, "programs")

		assert.Len(t, options, 2)
		assert.Contains(t, options, "Alpha")
		assert.Contains(t, options, "Beta")
	})

	t.Run("extracts and deduplicates options from multiple users", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":    "user1@example.com",
				"programs": []interface{}{"Alpha", "Beta"},
			},
			{
				"email":    "user2@example.com",
				"programs": []interface{}{"Beta", "Gamma"},
			},
			{
				"email":    "user3@example.com",
				"programs": []interface{}{"Alpha", "Delta"},
			},
		}

		options := extractMultiselectOptions(users, "programs")

		assert.Len(t, options, 4, "Should have 4 unique options")
		assert.Contains(t, options, "Alpha")
		assert.Contains(t, options, "Beta")
		assert.Contains(t, options, "Gamma")
		assert.Contains(t, options, "Delta")
	})

	t.Run("handles users without the field", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":    "user1@example.com",
				"programs": []interface{}{"Alpha"},
			},
			{
				"email":      "user2@example.com",
				"department": "Engineering", // Different field
			},
			{
				"email":    "user3@example.com",
				"programs": []interface{}{"Beta"},
			},
		}

		options := extractMultiselectOptions(users, "programs")

		assert.Len(t, options, 2)
		assert.Contains(t, options, "Alpha")
		assert.Contains(t, options, "Beta")
	})

	t.Run("handles empty arrays", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":    "user1@example.com",
				"programs": []interface{}{},
			},
			{
				"email":    "user2@example.com",
				"programs": []interface{}{"Alpha"},
			},
		}

		options := extractMultiselectOptions(users, "programs")

		assert.Len(t, options, 1)
		assert.Contains(t, options, "Alpha")
	})

	t.Run("skips non-array values", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":    "user1@example.com",
				"programs": "not an array", // Wrong type
			},
			{
				"email":    "user2@example.com",
				"programs": []interface{}{"Alpha"},
			},
		}

		options := extractMultiselectOptions(users, "programs")

		assert.Len(t, options, 1)
		assert.Contains(t, options, "Alpha")
	})

	t.Run("skips non-string array elements", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":    "user@example.com",
				"programs": []interface{}{"Alpha", 123, "Beta", nil, "Gamma"},
			},
		}

		options := extractMultiselectOptions(users, "programs")

		assert.Len(t, options, 3, "Should only include string values")
		assert.Contains(t, options, "Alpha")
		assert.Contains(t, options, "Beta")
		assert.Contains(t, options, "Gamma")
		assert.NotContains(t, options, 123)
		assert.NotContains(t, options, nil)
	})

	t.Run("skips empty strings", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":    "user@example.com",
				"programs": []interface{}{"Alpha", "", "Beta", ""},
			},
		}

		options := extractMultiselectOptions(users, "programs")

		assert.Len(t, options, 2, "Should skip empty strings")
		assert.Contains(t, options, "Alpha")
		assert.Contains(t, options, "Beta")
	})

	t.Run("returns empty slice for empty users array", func(t *testing.T) {
		users := []map[string]interface{}{}

		options := extractMultiselectOptions(users, "programs")

		assert.Empty(t, options)
	})

	t.Run("returns empty slice when no users have the field", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":      "user1@example.com",
				"department": "Engineering",
			},
			{
				"email":    "user2@example.com",
				"location": "US-East",
			},
		}

		options := extractMultiselectOptions(users, "nonexistent_field")

		assert.Empty(t, options)
	})

	t.Run("deduplicates duplicate values within single user", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":    "user@example.com",
				"programs": []interface{}{"Alpha", "Beta", "Alpha", "Beta"},
			},
		}

		options := extractMultiselectOptions(users, "programs")

		assert.Len(t, options, 2, "Should deduplicate within single user")
		assert.Contains(t, options, "Alpha")
		assert.Contains(t, options, "Beta")
	})

	t.Run("handles single option", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":    "user@example.com",
				"programs": []interface{}{"Alpha"},
			},
		}

		options := extractMultiselectOptions(users, "programs")

		assert.Len(t, options, 1)
		assert.Equal(t, "Alpha", options[0])
	})
}

func TestMergeOptions(t *testing.T) {
	t.Run("preserves existing option IDs", func(t *testing.T) {
		existingOptions := []map[string]interface{}{
			{"id": "id1", "name": "Alpha"},
			{"id": "id2", "name": "Beta"},
		}
		newValues := []string{"Alpha", "Beta"}

		merged, newCount := mergeOptions(existingOptions, newValues)

		assert.Len(t, merged, 2, "Should have same number of options")
		assert.Equal(t, 0, newCount, "No new options should be added")
		// Verify IDs are preserved
		assert.Equal(t, "id1", merged[0]["id"])
		assert.Equal(t, "Alpha", merged[0]["name"])
		assert.Equal(t, "id2", merged[1]["id"])
		assert.Equal(t, "Beta", merged[1]["name"])
	})

	t.Run("adds new options with new IDs", func(t *testing.T) {
		existingOptions := []map[string]interface{}{
			{"id": "id1", "name": "Alpha"},
		}
		newValues := []string{"Alpha", "Beta", "Gamma"}

		merged, newCount := mergeOptions(existingOptions, newValues)

		assert.Len(t, merged, 3, "Should have 3 options total")
		assert.Equal(t, 2, newCount, "Should have added 2 new options")

		// First option should be preserved
		assert.Equal(t, "id1", merged[0]["id"])
		assert.Equal(t, "Alpha", merged[0]["name"])

		// New options should have generated IDs
		assert.NotEmpty(t, merged[1]["id"])
		assert.Equal(t, "Beta", merged[1]["name"])
		assert.NotEmpty(t, merged[2]["id"])
		assert.Equal(t, "Gamma", merged[2]["name"])

		// New IDs should be different
		assert.NotEqual(t, merged[1]["id"], merged[2]["id"])
	})

	t.Run("append-only strategy keeps all existing options", func(t *testing.T) {
		existingOptions := []map[string]interface{}{
			{"id": "id1", "name": "Alpha"},
			{"id": "id2", "name": "Beta"},
			{"id": "id3", "name": "Gamma"},
		}
		// New values don't include "Beta" - but it should still be kept
		newValues := []string{"Alpha", "Gamma", "Delta"}

		merged, newCount := mergeOptions(existingOptions, newValues)

		assert.Len(t, merged, 4, "Should keep all existing + add new")
		assert.Equal(t, 1, newCount, "Should have added 1 new option")

		// All existing options should still be present
		names := make([]string, len(merged))
		for i, opt := range merged {
			names[i] = opt["name"].(string)
		}
		assert.Contains(t, names, "Alpha")
		assert.Contains(t, names, "Beta", "Beta should be kept even though not in new values")
		assert.Contains(t, names, "Gamma")
		assert.Contains(t, names, "Delta")
	})

	t.Run("handles empty existing options", func(t *testing.T) {
		existingOptions := []map[string]interface{}{}
		newValues := []string{"Alpha", "Beta"}

		merged, newCount := mergeOptions(existingOptions, newValues)

		assert.Len(t, merged, 2)
		assert.Equal(t, 2, newCount, "All values are new")
		assert.Equal(t, "Alpha", merged[0]["name"])
		assert.Equal(t, "Beta", merged[1]["name"])
	})

	t.Run("handles empty new values", func(t *testing.T) {
		existingOptions := []map[string]interface{}{
			{"id": "id1", "name": "Alpha"},
			{"id": "id2", "name": "Beta"},
		}
		newValues := []string{}

		merged, newCount := mergeOptions(existingOptions, newValues)

		assert.Len(t, merged, 2, "Should keep all existing")
		assert.Equal(t, 0, newCount, "No new options added")
		assert.Equal(t, "id1", merged[0]["id"])
		assert.Equal(t, "id2", merged[1]["id"])
	})

	t.Run("handles both empty", func(t *testing.T) {
		existingOptions := []map[string]interface{}{}
		newValues := []string{}

		merged, newCount := mergeOptions(existingOptions, newValues)

		assert.Empty(t, merged)
		assert.Equal(t, 0, newCount)
	})

	t.Run("deduplicates within new values", func(t *testing.T) {
		existingOptions := []map[string]interface{}{
			{"id": "id1", "name": "Alpha"},
		}
		// Duplicate "Beta" in new values
		newValues := []string{"Alpha", "Beta", "Beta", "Gamma"}

		merged, newCount := mergeOptions(existingOptions, newValues)

		assert.Len(t, merged, 3, "Beta should only be added once")
		assert.Equal(t, 2, newCount, "Should count Beta and Gamma as new")

		names := make([]string, len(merged))
		for i, opt := range merged {
			names[i] = opt["name"].(string)
		}
		assert.Equal(t, 1, countOccurrences(names, "Beta"), "Beta should appear exactly once")
	})

	t.Run("handles malformed existing options gracefully", func(t *testing.T) {
		existingOptions := []map[string]interface{}{
			{"id": "id1", "name": "Alpha"},
			{"id": 123, "name": "Beta"}, // Wrong type for id
			{"id": "id3", "name": 456},  // Wrong type for name
			{"name": "Delta"},           // Missing id
			{"id": "id5"},               // Missing name
		}
		newValues := []string{"Alpha", "Beta", "Gamma"}

		merged, _ := mergeOptions(existingOptions, newValues)

		// All existing options should be copied regardless of validity
		assert.Len(t, merged, 7, "Should preserve all existing + add valid new")

		// Gamma should be added (Beta exists but with malformed data, so might be re-added)
		names := make([]string, 0)
		for _, opt := range merged {
			if name, ok := opt["name"].(string); ok {
				names = append(names, name)
			}
		}
		assert.Contains(t, names, "Alpha")
		assert.Contains(t, names, "Gamma")
	})

	t.Run("generates valid IDs for new options", func(t *testing.T) {
		existingOptions := []map[string]interface{}{}
		newValues := []string{"Alpha", "Beta"}

		merged, _ := mergeOptions(existingOptions, newValues)

		for _, opt := range merged {
			id, ok := opt["id"].(string)
			assert.True(t, ok, "ID should be a string")
			assert.NotEmpty(t, id, "ID should not be empty")
			assert.True(t, model.IsValidId(id), "ID should be valid Mattermost ID")
		}
	})

	t.Run("maintains order of existing options", func(t *testing.T) {
		existingOptions := []map[string]interface{}{
			{"id": "id1", "name": "Zulu"},
			{"id": "id2", "name": "Alpha"},
			{"id": "id3", "name": "Mike"},
		}
		newValues := []string{"Bravo"}

		merged, _ := mergeOptions(existingOptions, newValues)

		// First 3 should maintain original order
		assert.Equal(t, "Zulu", merged[0]["name"])
		assert.Equal(t, "Alpha", merged[1]["name"])
		assert.Equal(t, "Mike", merged[2]["name"])
		assert.Equal(t, "Bravo", merged[3]["name"])
	})
}

// Helper function to count occurrences of a string in a slice
func countOccurrences(slice []string, target string) int {
	count := 0
	for _, item := range slice {
		if item == target {
			count++
		}
	}
	return count
}

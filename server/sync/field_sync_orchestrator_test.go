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

func TestSyncFields(t *testing.T) {
	groupID := "test-group-id"

	t.Run("creates new fields on first sync", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		users := []map[string]interface{}{
			{
				"email":      "user@example.com",
				"department": "Engineering",
				"location":   "US-East",
			},
		}

		// Mock KVStore - no existing fields
		cache.On("GetFieldID", "department").Return("", nil)
		cache.On("GetFieldID", "location").Return("", nil)

		// Mock field creation
		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.Name == "Department"
		})).Return(&model.PropertyField{ID: "dept-id", Name: "Department"}, nil)

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.Name == "Location"
		})).Return(&model.PropertyField{ID: "loc-id", Name: "Location"}, nil)

		// Mock KVStore saves
		cache.On("SaveFieldMapping", "department", "dept-id").Return(nil)
		cache.On("SaveFieldMapping", "location", "loc-id").Return(nil)

		mapping, err := SyncFields(client, groupID, users, cache)

		require.NoError(t, err)
		assert.Len(t, mapping, 2)
		assert.Equal(t, "dept-id", mapping["department"])
		assert.Equal(t, "loc-id", mapping["location"])
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("reuses existing fields", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		users := []map[string]interface{}{
			{
				"email":      "user@example.com",
				"department": "Engineering",
			},
		}

		// Mock KVStore - field exists
		cache.On("GetFieldID", "department").Return("existing-dept-id", nil)

		// No field creation should occur
		mapping, err := SyncFields(client, groupID, users, cache)

		require.NoError(t, err)
		assert.Len(t, mapping, 1)
		assert.Equal(t, "existing-dept-id", mapping["department"])
		api.AssertNotCalled(t, "CreatePropertyField")
		cache.AssertExpectations(t)
	})

	t.Run("creates multiselect field with options", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		users := []map[string]interface{}{
			{
				"email":    "user1@example.com",
				"programs": []interface{}{"Alpha", "Beta"},
			},
			{
				"email":    "user2@example.com",
				"programs": []interface{}{"Beta", "Gamma"},
			},
		}

		cache.On("GetFieldID", "programs").Return("", nil)

		// Mock field creation with options
		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			if f.Type != model.PropertyFieldTypeMultiselect {
				return false
			}
			options, ok := f.Attrs[model.PropertyFieldAttributeOptions].([]interface{})
			return ok && len(options) == 3 // Alpha, Beta, Gamma
		})).Return(&model.PropertyField{ID: "programs-id"}, nil)

		cache.On("SaveFieldMapping", "programs", "programs-id").Return(nil)
		cache.On("SaveFieldOptions", "programs", mock.Anything).Return(nil)

		// Mock logging (may log warnings if things fail)
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

		mapping, err := SyncFields(client, groupID, users, cache)

		require.NoError(t, err)
		assert.Equal(t, "programs-id", mapping["programs"])
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("updates existing multiselect field with new options", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		users := []map[string]interface{}{
			{
				"email":    "user@example.com",
				"programs": []interface{}{"Alpha", "Beta", "Gamma"},
			},
		}

		cache.On("GetFieldID", "programs").Return("existing-programs-id", nil)

		// Mock getting existing field
		existingField := &model.PropertyField{
			ID:      "existing-programs-id",
			GroupID: groupID,
			Type:    model.PropertyFieldTypeMultiselect,
			Attrs: model.StringInterface{
				model.PropertyFieldAttributeOptions: []interface{}{
					map[string]interface{}{"id": "alpha-id", "name": "Alpha"},
					map[string]interface{}{"id": "beta-id", "name": "Beta"},
				},
			},
		}
		api.On("GetPropertyField", groupID, "existing-programs-id").Return(existingField, nil)

		// Mock field update (should add Gamma)
		api.On("UpdatePropertyField", groupID, mock.MatchedBy(func(f *model.PropertyField) bool {
			options, ok := f.Attrs[model.PropertyFieldAttributeOptions].([]interface{})
			return ok && len(options) == 3 // Alpha, Beta, + Gamma
		})).Return(existingField, nil)

		cache.On("SaveFieldOptions", "programs", mock.Anything).Return(nil)

		// Mock logging (updateMultiselectOptions logs when it adds options)
		api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

		mapping, err := SyncFields(client, groupID, users, cache)

		require.NoError(t, err)
		assert.Equal(t, "existing-programs-id", mapping["programs"])
		api.AssertExpectations(t)
		cache.AssertExpectations(t)
	})

	t.Run("skips multiselect update when no new options", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		users := []map[string]interface{}{
			{
				"email":    "user@example.com",
				"programs": []interface{}{"Alpha", "Beta"},
			},
		}

		cache.On("GetFieldID", "programs").Return("existing-programs-id", nil)

		// Mock getting existing field with all options already present
		existingField := &model.PropertyField{
			ID:      "existing-programs-id",
			GroupID: groupID,
			Type:    model.PropertyFieldTypeMultiselect,
			Attrs: model.StringInterface{
				model.PropertyFieldAttributeOptions: []interface{}{
					map[string]interface{}{"id": "alpha-id", "name": "Alpha"},
					map[string]interface{}{"id": "beta-id", "name": "Beta"},
				},
			},
		}
		api.On("GetPropertyField", groupID, "existing-programs-id").Return(existingField, nil)

		// UpdatePropertyField should NOT be called (no new options)
		mapping, err := SyncFields(client, groupID, users, cache)

		require.NoError(t, err)
		assert.Equal(t, "existing-programs-id", mapping["programs"])
		api.AssertNotCalled(t, "UpdatePropertyField")
		cache.AssertExpectations(t)
	})

	t.Run("handles partial failures gracefully", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		users := []map[string]interface{}{
			{
				"email":      "user@example.com",
				"department": "Engineering",
				"location":   "US-East",
			},
		}

		cache.On("GetFieldID", "department").Return("", nil)
		cache.On("GetFieldID", "location").Return("", nil)

		// Department creation fails
		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.Name == "Department"
		})).Return(nil, errors.New("API error"))

		// Mock fallback SearchPropertyFields call (department doesn't exist)
		api.On("SearchPropertyFields", groupID, mock.Anything).Return([]*model.PropertyField{}, nil)

		// Location creation succeeds
		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.Name == "Location"
		})).Return(&model.PropertyField{ID: "loc-id"}, nil)

		cache.On("SaveFieldMapping", "location", "loc-id").Return(nil)

		// Mock logging (will log error for department failure)
		api.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

		mapping, err := SyncFields(client, groupID, users, cache)

		// Should not return error (graceful degradation)
		require.NoError(t, err)
		// Should only have location (department failed)
		assert.Len(t, mapping, 1)
		assert.Equal(t, "loc-id", mapping["location"])
		assert.NotContains(t, mapping, "department")
	})

	t.Run("handles empty users", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		users := []map[string]interface{}{}

		mapping, err := SyncFields(client, groupID, users, cache)

		require.NoError(t, err)
		assert.Empty(t, mapping)
		api.AssertNotCalled(t, "CreatePropertyField")
	})

	t.Run("handles mixed new and existing fields", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		users := []map[string]interface{}{
			{
				"email":      "user@example.com",
				"department": "Engineering", // Existing
				"location":   "US-East",     // New
			},
		}

		// Department exists, location is new
		cache.On("GetFieldID", "department").Return("existing-dept-id", nil)
		cache.On("GetFieldID", "location").Return("", nil)

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.Name == "Location"
		})).Return(&model.PropertyField{ID: "loc-id"}, nil)

		cache.On("SaveFieldMapping", "location", "loc-id").Return(nil)

		mapping, err := SyncFields(client, groupID, users, cache)

		require.NoError(t, err)
		assert.Len(t, mapping, 2)
		assert.Equal(t, "existing-dept-id", mapping["department"])
		assert.Equal(t, "loc-id", mapping["location"])
	})

	t.Run("excludes email field", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		cache := &mockFieldCache{}

		users := []map[string]interface{}{
			{
				"email":      "user@example.com",
				"department": "Engineering",
			},
		}

		cache.On("GetFieldID", "department").Return("dept-id", nil)

		mapping, err := SyncFields(client, groupID, users, cache)

		require.NoError(t, err)
		assert.Len(t, mapping, 1)
		assert.Contains(t, mapping, "department")
		assert.NotContains(t, mapping, "email")
	})
}

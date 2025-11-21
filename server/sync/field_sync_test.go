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

	t.Run("creates all hardcoded fields", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		// Mock field creation for each hardcoded field
		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.ID == FieldIDJobTitle
		})).Return(&model.PropertyField{ID: FieldIDJobTitle, Name: "Job Title"}, nil)

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.ID == FieldIDPrograms
		})).Return(&model.PropertyField{ID: FieldIDPrograms, Name: "Programs"}, nil)

		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			return f.ID == FieldIDStartDate
		})).Return(&model.PropertyField{ID: FieldIDStartDate, Name: "Start Date"}, nil)

		// Mock logging
		api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

		err := SyncFields(client, groupID)

		require.NoError(t, err)
		api.AssertExpectations(t)
	})

	t.Run("updates existing fields", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		// Simulate fields already existing (creation returns error)
		api.On("CreatePropertyField", mock.Anything).Return(nil, errors.New("duplicate key"))

		// Mock GetPropertyField for each field (simulating they exist)
		for _, def := range fieldDefinitions {
			existingField := &model.PropertyField{
				ID:      def.ID,
				GroupID: groupID,
				Name:    def.Name,
				Type:    def.Type,
				Attrs:   make(model.StringInterface),
			}
			api.On("GetPropertyField", groupID, def.ID).Return(existingField, nil).Once()
			api.On("UpdatePropertyField", groupID, mock.MatchedBy(func(f *model.PropertyField) bool {
				return f.ID == def.ID
			})).Return(existingField, nil).Once()
		}

		// Mock logging
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

		err := SyncFields(client, groupID)

		require.NoError(t, err)
		api.AssertExpectations(t)
	})

	t.Run("multiselect field includes options", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		// Mock field creation - verify Programs field has options
		api.On("CreatePropertyField", mock.MatchedBy(func(f *model.PropertyField) bool {
			if f.ID != FieldIDPrograms {
				return true // Accept other fields
			}
			// Verify Programs field has correct options
			options, ok := f.Attrs[model.PropertyFieldAttributeOptions].([]interface{})
			if !ok {
				return false
			}
			// Should have 3 options: Apples, Oranges, Lemons
			if len(options) != 3 {
				return false
			}
			return true
		})).Return(&model.PropertyField{ID: FieldIDPrograms}, nil)

		// Mock logging
		api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

		err := SyncFields(client, groupID)

		require.NoError(t, err)
		api.AssertExpectations(t)
	})

	t.Run("continues on partial failures", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		callCount := 0
		// First field fails, others succeed
		api.On("CreatePropertyField", mock.Anything).Return(func(f *model.PropertyField) (*model.PropertyField, error) {
			callCount++
			if callCount == 1 {
				// First call fails
				return nil, errors.New("API error")
			}
			// Subsequent calls succeed
			return f, nil
		})

		// Mock GetPropertyField for the failed field (doesn't exist)
		api.On("GetPropertyField", groupID, mock.Anything).Return(nil, errors.New("not found")).Once()

		// Mock logging
		api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		api.On("LogDebug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		api.On("LogError", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
		api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()

		err := SyncFields(client, groupID)

		// Should not return error (graceful degradation)
		require.NoError(t, err)
	})
}

func TestGetFieldID(t *testing.T) {
	t.Run("returns correct field IDs", func(t *testing.T) {
		assert.Equal(t, FieldIDJobTitle, GetFieldID("job_title"))
		assert.Equal(t, FieldIDPrograms, GetFieldID("programs"))
		assert.Equal(t, FieldIDStartDate, GetFieldID("start_date"))
	})

	t.Run("returns empty string for unknown field", func(t *testing.T) {
		assert.Equal(t, "", GetFieldID("unknown_field"))
	})
}

func TestGetProgramOptionID(t *testing.T) {
	t.Run("returns correct option IDs", func(t *testing.T) {
		assert.Equal(t, OptionIDApples, GetProgramOptionID("Apples"))
		assert.Equal(t, OptionIDOranges, GetProgramOptionID("Oranges"))
		assert.Equal(t, OptionIDLemons, GetProgramOptionID("Lemons"))
	})

	t.Run("returns empty string for unknown option", func(t *testing.T) {
		assert.Equal(t, "", GetProgramOptionID("Unknown"))
	})
}

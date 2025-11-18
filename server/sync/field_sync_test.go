package sync

import (
	"errors"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockKVStore is a mock implementation of the KVStore interface for testing
type mockKVStore struct {
	mock.Mock
}

func (m *mockKVStore) SaveFieldMapping(fieldName, fieldID string) error {
	args := m.Called(fieldName, fieldID)
	return args.Error(0)
}

func (m *mockKVStore) GetFieldMapping(fieldName string) (string, error) {
	args := m.Called(fieldName)
	return args.String(0), args.Error(1)
}

func (m *mockKVStore) SaveFieldOptions(fieldName string, options map[string]string) error {
	args := m.Called(fieldName, options)
	return args.Error(0)
}

func (m *mockKVStore) GetFieldOptions(fieldName string) (map[string]string, error) {
	args := m.Called(fieldName)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *mockKVStore) SaveLastSyncTime(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *mockKVStore) GetLastSyncTime() (time.Time, error) {
	args := m.Called()
	return args.Get(0).(time.Time), args.Error(1)
}

func TestCreatePropertyField(t *testing.T) {
	groupID := "test-group-id"

	t.Run("successfully creates text field", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		store := &mockKVStore{}

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

		store.On("SaveFieldMapping", "department", "field-id-123").Return(nil)

		result, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, store)

		require.NoError(t, err)
		assert.Equal(t, "field-id-123", result.ID)
		assert.Equal(t, "Department", result.Name)
		assert.Equal(t, model.PropertyFieldTypeText, result.Type)
		api.AssertExpectations(t)
		store.AssertExpectations(t)
	})

	t.Run("successfully creates multiselect field", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		store := &mockKVStore{}

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
		store.On("SaveFieldMapping", "security_clearance", "field-id-456").Return(nil)

		result, err := createPropertyField(client, groupID, "security_clearance", model.PropertyFieldTypeMultiselect, store)

		require.NoError(t, err)
		assert.Equal(t, "field-id-456", result.ID)
		assert.Equal(t, model.PropertyFieldTypeMultiselect, result.Type)
		api.AssertExpectations(t)
		store.AssertExpectations(t)
	})

	t.Run("successfully creates date field", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		store := &mockKVStore{}

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
		store.On("SaveFieldMapping", "start_date", "field-id-789").Return(nil)

		result, err := createPropertyField(client, groupID, "start_date", model.PropertyFieldTypeDate, store)

		require.NoError(t, err)
		assert.Equal(t, "field-id-789", result.ID)
		assert.Equal(t, model.PropertyFieldTypeDate, result.Type)
		api.AssertExpectations(t)
		store.AssertExpectations(t)
	})

	t.Run("uses display name transformation", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		store := &mockKVStore{}

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

		store.On("SaveFieldMapping", "user_access_level", "field-id").Return(nil)

		result, err := createPropertyField(client, groupID, "user_access_level", model.PropertyFieldTypeText, store)

		require.NoError(t, err)
		assert.Equal(t, "User Access Level", result.Name)
		api.AssertExpectations(t)
		store.AssertExpectations(t)
	})

	t.Run("sets correct visibility attribute", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		store := &mockKVStore{}

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

		store.On("SaveFieldMapping", "department", "field-id").Return(nil)

		_, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, store)

		require.NoError(t, err)
		api.AssertExpectations(t)
		store.AssertExpectations(t)
	})

	t.Run("sets managed attribute to admin", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		store := &mockKVStore{}

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

		store.On("SaveFieldMapping", "department", "field-id").Return(nil)

		_, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, store)

		require.NoError(t, err)
		api.AssertExpectations(t)
		store.AssertExpectations(t)
	})

	t.Run("returns error when API call fails", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		store := &mockKVStore{}

		apiError := errors.New("API error: insufficient permissions")
		api.On("CreatePropertyField", mock.Anything).Return(nil, apiError)

		result, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, store)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to create property field")
		api.AssertExpectations(t)
		// KVStore should not be called if API fails
		store.AssertNotCalled(t, "SaveFieldMapping")
	})

	t.Run("returns field but logs error when KVStore save fails", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})
		store := &mockKVStore{}

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
		store.On("SaveFieldMapping", "department", "field-id-123").Return(kvError)

		result, err := createPropertyField(client, groupID, "department", model.PropertyFieldTypeText, store)

		// Field should still be returned even though KVStore save failed
		require.Error(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "field-id-123", result.ID)
		assert.Contains(t, err.Error(), "failed to save mapping")
		api.AssertExpectations(t)
		store.AssertExpectations(t)
	})
}

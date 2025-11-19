package sync

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatStringValue(t *testing.T) {
	t.Run("simple text value", func(t *testing.T) {
		result, err := formatStringValue("Engineering")
		require.NoError(t, err)

		// Verify it's properly JSON-encoded (with quotes)
		assert.Equal(t, json.RawMessage(`"Engineering"`), result)

		// Verify it unmarshals correctly
		var decoded string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "Engineering", decoded)
	})

	t.Run("date string value", func(t *testing.T) {
		result, err := formatStringValue("2023-01-15")
		require.NoError(t, err)

		// Verify it's properly JSON-encoded
		assert.Equal(t, json.RawMessage(`"2023-01-15"`), result)

		var decoded string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "2023-01-15", decoded)
	})

	t.Run("empty string", func(t *testing.T) {
		result, err := formatStringValue("")
		require.NoError(t, err)

		// Empty string should be encoded as ""
		assert.Equal(t, json.RawMessage(`""`), result)

		var decoded string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "", decoded)
	})

	t.Run("string with special characters", func(t *testing.T) {
		result, err := formatStringValue(`He said "hello"`)
		require.NoError(t, err)

		// Quotes should be escaped
		var decoded string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, `He said "hello"`, decoded)
	})

	t.Run("string with newlines and tabs", func(t *testing.T) {
		result, err := formatStringValue("Line 1\nLine 2\tTabbed")
		require.NoError(t, err)

		// Special characters should be escaped
		var decoded string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "Line 1\nLine 2\tTabbed", decoded)
	})

	t.Run("string with backslashes", func(t *testing.T) {
		result, err := formatStringValue(`C:\Users\John`)
		require.NoError(t, err)

		// Backslashes should be escaped
		var decoded string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, `C:\Users\John`, decoded)
	})

	t.Run("unicode characters", func(t *testing.T) {
		result, err := formatStringValue("Hello 世界 🌍")
		require.NoError(t, err)

		var decoded string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, "Hello 世界 🌍", decoded)
	})
}

func TestFormatMultiselectValue(t *testing.T) {
	t.Run("multiple option values", func(t *testing.T) {
		cache := &mockFieldCache{}
		cache.On("GetOptionID", "security_clearance", "Level1").Return("opt_abc123", nil)
		cache.On("GetOptionID", "security_clearance", "Level3").Return("opt_ghi789", nil)

		result, err := formatMultiselectValue(cache, "security_clearance", []string{"Level1", "Level3"})
		require.NoError(t, err)

		// Verify it's properly JSON-encoded array of IDs
		var decoded []string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, []string{"opt_abc123", "opt_ghi789"}, decoded)

		cache.AssertExpectations(t)
	})

	t.Run("single option value", func(t *testing.T) {
		cache := &mockFieldCache{}
		cache.On("GetOptionID", "programs", "Apples").Return("opt_aaa111", nil)

		result, err := formatMultiselectValue(cache, "programs", []string{"Apples"})
		require.NoError(t, err)

		var decoded []string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, []string{"opt_aaa111"}, decoded)

		cache.AssertExpectations(t)
	})

	t.Run("empty array", func(t *testing.T) {
		cache := &mockFieldCache{}

		result, err := formatMultiselectValue(cache, "programs", []string{})
		require.NoError(t, err)

		// Empty array should be encoded as []
		assert.Equal(t, json.RawMessage(`[]`), result)

		var decoded []string
		err = json.Unmarshal(result, &decoded)
		require.NoError(t, err)
		assert.Equal(t, []string{}, decoded)

		cache.AssertExpectations(t)
	})

	t.Run("missing option returns error", func(t *testing.T) {
		cache := &mockFieldCache{}
		cache.On("GetOptionID", "security_clearance", "Level1").Return("opt_abc123", nil)
		cache.On("GetOptionID", "security_clearance", "Level99").Return("", assert.AnError)

		_, err := formatMultiselectValue(cache, "security_clearance", []string{"Level1", "Level99"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Level99")
		assert.Contains(t, err.Error(), "security_clearance")

		cache.AssertExpectations(t)
	})

	t.Run("missing field returns error", func(t *testing.T) {
		cache := &mockFieldCache{}
		cache.On("GetOptionID", "nonexistent_field", "Value1").Return("", assert.AnError)

		_, err := formatMultiselectValue(cache, "nonexistent_field", []string{"Value1"})
		require.Error(t, err)

		cache.AssertExpectations(t)
	})

	t.Run("empty option ID returns error", func(t *testing.T) {
		cache := &mockFieldCache{}
		cache.On("GetOptionID", "programs", "Oranges").Return("", nil)

		_, err := formatMultiselectValue(cache, "programs", []string{"Oranges"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Oranges")
		assert.Contains(t, err.Error(), "not found")

		cache.AssertExpectations(t)
	})
}

func TestBuildPropertyValues(t *testing.T) {
	groupID := "test-group-id"
	user := &model.User{
		Id:    "user123",
		Email: "test@example.com",
	}

	t.Run("builds values for all field types", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		cache := &mockFieldCache{}
		cache.On("GetFieldID", "department").Return("field_dept", nil)
		cache.On("GetFieldID", "start_date").Return("field_date", nil)
		cache.On("GetFieldID", "programs").Return("field_prog", nil)
		cache.On("GetOptionID", "programs", "Apples").Return("opt_apple", nil)
		cache.On("GetOptionID", "programs", "Oranges").Return("opt_orange", nil)

		userAttrs := map[string]interface{}{
			"email":      "test@example.com", // Should be skipped
			"department": "Engineering",
			"start_date": "2023-01-15",
			"programs":   []interface{}{"Apples", "Oranges"},
		}

		values, err := buildPropertyValues(client, user, groupID, userAttrs, cache)
		require.NoError(t, err)
		assert.Len(t, values, 3) // email excluded

		// Verify all values have correct structure
		for _, v := range values {
			assert.Equal(t, groupID, v.GroupID)
			assert.Equal(t, "user", v.TargetType)
			assert.Equal(t, user.Id, v.TargetID)
			assert.NotEmpty(t, v.FieldID)
			assert.NotEmpty(t, v.Value)
		}

		cache.AssertExpectations(t)
	})

	t.Run("handles string array for multiselect", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		cache := &mockFieldCache{}
		cache.On("GetFieldID", "tags").Return("field_tags", nil)
		cache.On("GetOptionID", "tags", "Tag1").Return("opt_tag1", nil)
		cache.On("GetOptionID", "tags", "Tag2").Return("opt_tag2", nil)

		userAttrs := map[string]interface{}{
			"email": "test@example.com",
			"tags":  []string{"Tag1", "Tag2"},
		}

		values, err := buildPropertyValues(client, user, groupID, userAttrs, cache)
		require.NoError(t, err)
		assert.Len(t, values, 1)

		// Verify multiselect was formatted correctly
		var optionIDs []string
		err = json.Unmarshal(values[0].Value, &optionIDs)
		require.NoError(t, err)
		assert.Equal(t, []string{"opt_tag1", "opt_tag2"}, optionIDs)

		cache.AssertExpectations(t)
	})

	t.Run("skips email field", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		cache := &mockFieldCache{}

		userAttrs := map[string]interface{}{
			"email": "test@example.com",
		}

		values, err := buildPropertyValues(client, user, groupID, userAttrs, cache)
		require.NoError(t, err)
		assert.Len(t, values, 0)

		cache.AssertExpectations(t)
	})

	t.Run("skips field with missing field ID", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		cache := &mockFieldCache{}
		cache.On("GetFieldID", "unknown_field").Return("", assert.AnError)
		cache.On("GetFieldID", "department").Return("field_dept", nil)

		// Expect log warning for unknown field
		api.On("LogWarn", "Failed to get field ID, skipping field",
			"field_name", "unknown_field",
			"user_email", "test@example.com",
			"error", assert.AnError.Error())

		userAttrs := map[string]interface{}{
			"email":         "test@example.com",
			"unknown_field": "value",
			"department":    "Engineering",
		}

		values, err := buildPropertyValues(client, user, groupID, userAttrs, cache)
		require.NoError(t, err)
		assert.Len(t, values, 1) // Only department

		cache.AssertExpectations(t)
	})

	t.Run("skips field with unsupported type", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		cache := &mockFieldCache{}
		cache.On("GetFieldID", "bad_field").Return("field_bad", nil)
		cache.On("GetFieldID", "department").Return("field_dept", nil)

		// Expect log warning for unsupported type
		api.On("LogWarn", "Unsupported field value type, skipping field",
			"field_name", "bad_field",
			"user_email", "test@example.com",
			"value_type", "int")

		userAttrs := map[string]interface{}{
			"email":      "test@example.com",
			"bad_field":  123, // Unsupported type
			"department": "Engineering",
		}

		values, err := buildPropertyValues(client, user, groupID, userAttrs, cache)
		require.NoError(t, err)
		assert.Len(t, values, 1) // Only department

		cache.AssertExpectations(t)
	})

	t.Run("skips field with format error", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		cache := &mockFieldCache{}
		cache.On("GetFieldID", "programs").Return("field_prog", nil)
		cache.On("GetOptionID", "programs", "InvalidOption").Return("", assert.AnError)
		cache.On("GetFieldID", "department").Return("field_dept", nil)

		// Expect log warning for format error
		api.On("LogWarn", "Failed to format field value, skipping field",
			"field_name", "programs",
			"user_email", "test@example.com",
			"error", "failed to get option ID for programs.InvalidOption: assert.AnError general error for testing")

		userAttrs := map[string]interface{}{
			"email":      "test@example.com",
			"programs":   []string{"InvalidOption"},
			"department": "Engineering",
		}

		values, err := buildPropertyValues(client, user, groupID, userAttrs, cache)
		require.NoError(t, err)
		assert.Len(t, values, 1) // Only department

		cache.AssertExpectations(t)
	})

	t.Run("handles empty attributes", func(t *testing.T) {
		api := &plugintest.API{}
		client := pluginapi.NewClient(api, &plugintest.Driver{})

		cache := &mockFieldCache{}

		userAttrs := map[string]interface{}{}

		values, err := buildPropertyValues(client, user, groupID, userAttrs, cache)
		require.NoError(t, err)
		assert.Len(t, values, 0)

		cache.AssertExpectations(t)
	})
}

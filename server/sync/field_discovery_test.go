package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoverFields(t *testing.T) {
	t.Run("discovers fields from single user", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":      "user@example.com",
				"department": "Engineering",
				"location":   "US-East",
			},
		}

		fields := discoverFields(users)

		assert.Len(t, fields, 2, "Should discover 2 fields (excluding email)")
		assert.Equal(t, "Engineering", fields["department"])
		assert.Equal(t, "US-East", fields["location"])
		assert.NotContains(t, fields, "email", "Email should be excluded")
	})

	t.Run("discovers fields from multiple users", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":      "user1@example.com",
				"department": "Engineering",
				"location":   "US-East",
			},
			{
				"email":      "user2@example.com",
				"department": "Sales",
				"location":   "US-West",
			},
		}

		fields := discoverFields(users)

		assert.Len(t, fields, 2)
		assert.Contains(t, fields, "department")
		assert.Contains(t, fields, "location")
		assert.NotContains(t, fields, "email")
	})

	t.Run("handles varying fields across users (union)", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":      "user1@example.com",
				"department": "Engineering",
				"location":   "US-East",
			},
			{
				"email":           "user2@example.com",
				"department":      "Sales",
				"clearance_level": "Level2",
			},
			{
				"email":    "user3@example.com",
				"programs": []interface{}{"Alpha", "Beta"},
			},
		}

		fields := discoverFields(users)

		assert.Len(t, fields, 4, "Should discover union of all fields")
		assert.Contains(t, fields, "department")
		assert.Contains(t, fields, "location")
		assert.Contains(t, fields, "clearance_level")
		assert.Contains(t, fields, "programs")
		assert.NotContains(t, fields, "email")
	})

	t.Run("excludes email field", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":      "user@example.com",
				"department": "Engineering",
			},
		}

		fields := discoverFields(users)

		assert.NotContains(t, fields, "email", "Email field should always be excluded")
		assert.Contains(t, fields, "department")
	})

	t.Run("handles empty users array", func(t *testing.T) {
		users := []map[string]interface{}{}

		fields := discoverFields(users)

		assert.Empty(t, fields, "Should return empty map for empty users array")
	})

	t.Run("skips nil values", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":      "user1@example.com",
				"department": nil,
				"location":   "US-East",
			},
		}

		fields := discoverFields(users)

		assert.Len(t, fields, 1, "Should only include non-nil fields")
		assert.Equal(t, "US-East", fields["location"])
		assert.NotContains(t, fields, "department", "Nil values should be skipped")
	})

	t.Run("uses non-nil value when some users have nil", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":      "user1@example.com",
				"department": nil,
			},
			{
				"email":      "user2@example.com",
				"department": "Engineering",
			},
		}

		fields := discoverFields(users)

		assert.Len(t, fields, 1)
		assert.Equal(t, "Engineering", fields["department"], "Should use non-nil value from second user")
	})

	t.Run("skips field when all users have nil", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":      "user1@example.com",
				"department": nil,
			},
			{
				"email":      "user2@example.com",
				"department": nil,
			},
		}

		fields := discoverFields(users)

		assert.Empty(t, fields, "Should not discover fields with only nil values")
		assert.NotContains(t, fields, "department")
	})

	t.Run("handles different value types", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":       "user@example.com",
				"text_field":  "some text",
				"array_field": []interface{}{"value1", "value2"},
				"date_field":  "2023-01-15",
			},
		}

		fields := discoverFields(users)

		assert.Len(t, fields, 3)
		assert.Equal(t, "some text", fields["text_field"])
		assert.Equal(t, []interface{}{"value1", "value2"}, fields["array_field"])
		assert.Equal(t, "2023-01-15", fields["date_field"])
	})

	t.Run("handles user with only email field", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email": "user@example.com",
			},
		}

		fields := discoverFields(users)

		assert.Empty(t, fields, "Should return empty map when user only has email")
	})

	t.Run("preserves first non-nil sample value", func(t *testing.T) {
		users := []map[string]interface{}{
			{
				"email":      "user1@example.com",
				"department": "Engineering",
			},
			{
				"email":      "user2@example.com",
				"department": "Sales",
			},
		}

		fields := discoverFields(users)

		// Should preserve the first encountered value
		assert.Equal(t, "Engineering", fields["department"], "Should use first non-nil value encountered")
	})
}

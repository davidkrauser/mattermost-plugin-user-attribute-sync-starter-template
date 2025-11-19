package sync

import (
	"encoding/json"
	"testing"

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

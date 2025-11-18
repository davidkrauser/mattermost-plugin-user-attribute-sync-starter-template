package kvstore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// Key constants for storing sync-related data in the KVStore.
// These keys are used to persist state that must survive plugin restarts,
// enabling incremental synchronization and preventing duplicate field creation.
const (
	// fieldMappingPrefix is used to store the mapping from external field names
	// to Mattermost PropertyField IDs. This prevents creating duplicate fields
	// on subsequent syncs.
	// Format: "field_mapping_{fieldName}" -> fieldID
	fieldMappingPrefix = "field_mapping_"

	// fieldOptionsPrefix is used to store accumulated multiselect options for each field.
	// Options are stored internally as JSON but exposed as map[string]string (option name -> option ID).
	// This enables the append-only option management strategy.
	// Format: "field_options_{fieldName}" -> JSON-encoded map[string]string
	fieldOptionsPrefix = "field_options_"

	// lastSyncTimestampKey stores the timestamp of the last successful sync.
	// This enables incremental synchronization where only changed users are processed
	// after the first full sync.
	lastSyncTimestampKey = "last_sync_timestamp"
)

// SaveFieldMapping stores the mapping from a field name to its Mattermost PropertyField ID.
// This mapping is used to avoid creating duplicate fields on subsequent syncs and to
// quickly look up field IDs when syncing user values.
//
// Parameters:
//   - fieldName: The internal field name (e.g., "security_clearance")
//   - fieldID: The Mattermost PropertyField ID (e.g., "abc123xyz")
//
// Returns an error if the KVStore operation fails.
func (kv Client) SaveFieldMapping(fieldName, fieldID string) error {
	key := fieldMappingPrefix + fieldName
	_, err := kv.client.KV.Set(key, []byte(fieldID))
	if err != nil {
		return errors.Wrapf(err, "failed to save field mapping for %s", fieldName)
	}
	return nil
}

// GetFieldMapping retrieves the Mattermost PropertyField ID for a given field name.
// Returns an empty string if no mapping exists (field hasn't been created yet).
//
// Parameters:
//   - fieldName: The internal field name to look up
//
// Returns:
//   - The PropertyField ID if found, empty string otherwise
//   - Error if the KVStore operation fails (but not if key doesn't exist)
func (kv Client) GetFieldMapping(fieldName string) (string, error) {
	key := fieldMappingPrefix + fieldName
	var fieldID string
	err := kv.client.KV.Get(key, &fieldID)
	if err != nil {
		// Return empty string if key doesn't exist (not an error condition)
		return "", nil
	}
	return fieldID, nil
}

// SaveFieldOptions stores the accumulated multiselect options for a field.
// This is used to implement append-only option management where options are
// never removed, only added. JSON marshaling is handled internally.
//
// Parameters:
//   - fieldName: The field name
//   - options: Map of option name -> option ID
//
// Returns an error if JSON marshaling or KVStore operation fails.
func (kv Client) SaveFieldOptions(fieldName string, options map[string]string) error {
	key := fieldOptionsPrefix + fieldName

	// Marshal the options map to JSON
	data, err := json.Marshal(options)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal options for %s", fieldName)
	}

	_, err = kv.client.KV.Set(key, data)
	if err != nil {
		return errors.Wrapf(err, "failed to save field options for %s", fieldName)
	}
	return nil
}

// GetFieldOptions retrieves the accumulated multiselect options for a field.
// Returns an empty map if no options have been stored yet.
// JSON unmarshaling is handled internally.
//
// Parameters:
//   - fieldName: The field name to look up
//
// Returns:
//   - Map of option name -> option ID if found, empty map otherwise
//   - Error if the KVStore operation fails or JSON parsing fails
func (kv Client) GetFieldOptions(fieldName string) (map[string]string, error) {
	key := fieldOptionsPrefix + fieldName
	var data []byte
	err := kv.client.KV.Get(key, &data)
	if err != nil {
		// Return empty map if key doesn't exist (not an error condition)
		return make(map[string]string), nil
	}

	// If no data was stored, return empty map
	if len(data) == 0 {
		return make(map[string]string), nil
	}

	// Unmarshal the JSON data
	var options map[string]string
	err = json.Unmarshal(data, &options)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal options for %s", fieldName)
	}

	return options, nil
}

// SaveLastSyncTime stores the timestamp of the last successful sync.
// This timestamp is used by the AttributeProvider to determine which users
// have changed since the last sync, enabling incremental synchronization.
//
// Parameters:
//   - t: The timestamp to store (typically time.Now())
//
// Returns an error if the KVStore operation fails.
func (kv Client) SaveLastSyncTime(t time.Time) error {
	// Store as RFC3339 format for readability and easy parsing
	timestamp := t.Format(time.RFC3339)
	_, err := kv.client.KV.Set(lastSyncTimestampKey, []byte(timestamp))
	if err != nil {
		return errors.Wrap(err, "failed to save last sync timestamp")
	}
	return nil
}

// GetLastSyncTime retrieves the timestamp of the last successful sync.
// Returns zero time if no sync has been performed yet (first sync).
//
// Returns:
//   - The last sync timestamp if found, zero time otherwise
//   - Error if the KVStore operation fails or timestamp parsing fails
func (kv Client) GetLastSyncTime() (time.Time, error) {
	var timestamp string
	err := kv.client.KV.Get(lastSyncTimestampKey, &timestamp)
	if err != nil {
		// Return zero time if key doesn't exist (first sync)
		return time.Time{}, nil
	}

	// Parse the RFC3339 timestamp
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return time.Time{}, errors.Wrap(err, fmt.Sprintf("failed to parse timestamp: %s", timestamp))
	}

	return t, nil
}

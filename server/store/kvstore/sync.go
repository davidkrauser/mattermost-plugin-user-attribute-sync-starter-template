package kvstore

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// Key constant for storing sync-related data in the KVStore.
const (
	// lastSyncTimestampKey stores the timestamp of the last successful sync.
	// This enables incremental synchronization where only changed users are processed
	// after the first full sync.
	lastSyncTimestampKey = "last_sync_timestamp"
)

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

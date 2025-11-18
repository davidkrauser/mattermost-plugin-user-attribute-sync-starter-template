package kvstore

import "time"

type KVStore interface {
	// Field mapping methods - track external field names to Mattermost PropertyField IDs
	SaveFieldMapping(fieldName, fieldID string) error
	GetFieldMapping(fieldName string) (string, error)

	// Field options methods - store accumulated multiselect options
	SaveFieldOptions(fieldName string, options map[string]string) error
	GetFieldOptions(fieldName string) (map[string]string, error)

	// Sync timestamp methods - enable incremental synchronization
	SaveLastSyncTime(t time.Time) error
	GetLastSyncTime() (time.Time, error)
}

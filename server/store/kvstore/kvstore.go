package kvstore

import "time"

type KVStore interface {
	// Sync timestamp methods - enable incremental synchronization
	SaveLastSyncTime(t time.Time) error
	GetLastSyncTime() (time.Time, error)
}

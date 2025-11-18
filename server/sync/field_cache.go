package sync

import (
	"github.com/pkg/errors"

	"github.com/mattermost/user-attribute-sync-starter-template/server/store/kvstore"
)

// FieldCache provides in-memory caching of field mappings and multiselect options
// to optimize performance during value synchronization.
//
// Performance problem without cache:
// During value synchronization, each user's attributes need to be converted to PropertyValues.
// This requires looking up field IDs and option IDs from KVStore. For example:
//   - 100 users × 5 attributes = 500 field ID lookups
//   - 100 users × 2 multiselect fields × 3 options average = 600 option ID lookups
//   - Total: 1100+ KVStore reads per sync
//
// With cache (lazy-loading):
//   - First lookup: Read from KVStore and cache (5 unique fields = 5 reads)
//   - Subsequent lookups: Hit in-memory cache (495 cache hits, 0 KVStore reads)
//   - Result: ~100x reduction in KVStore operations
//
// Cache lifecycle:
// The cache has a per-sync lifecycle - it's created fresh for each sync run:
//  1. Sync starts: Create cache (empty)
//  2. Field sync: Updates cache via SaveFieldMapping/SaveFieldOptions (write-through)
//  3. Value sync: Reads from cache via GetFieldID/GetOptionID (lazy-load on cache miss)
//  4. Sync ends: Cache is discarded
//  5. Next sync: Fresh cache created
//
// Lazy-loading strategy:
// On cache miss, the implementation fetches from KVStore and caches the result.
// This means:
//   - Each unique field is loaded from KVStore at most once per sync
//   - No need for "Load all" operation (which KVStore doesn't support)
//   - Simple and efficient
//
// Write-through strategy:
// All writes (SaveFieldMapping, SaveFieldOptions) update both:
//  1. In-memory cache (for immediate reads during this sync)
//  2. KVStore (for persistence and next sync's lazy loads)
//
// This maintains consistency between cache and persistent storage.
//
// Interface design:
// The interface abstraction allows plugin developers to extend or replace the
// caching implementation if needed (e.g., Redis-backed cache for multi-server).
type FieldCache interface {
	// GetFieldID retrieves the field ID for a given field name.
	// On cache miss, fetches from KVStore and caches the result.
	// Returns empty string if field doesn't exist (not an error condition).
	// Returns error only on KVStore failures.
	GetFieldID(fieldName string) (string, error)

	// GetOptionID retrieves the option ID for a given field and option name.
	// On cache miss, fetches field's options from KVStore and caches them.
	// Returns empty string if field or option doesn't exist (not an error condition).
	// Returns error only on KVStore failures.
	GetOptionID(fieldName, optionName string) (string, error)

	// SaveFieldMapping saves a field name → ID mapping to both cache and KVStore (write-through).
	// Returns error if KVStore write fails.
	SaveFieldMapping(fieldName, fieldID string) error

	// SaveFieldOptions saves option mappings for a field to both cache and KVStore (write-through).
	// Returns error if KVStore write fails.
	SaveFieldOptions(fieldName string, options map[string]string) error
}

// fieldCacheImpl is the default implementation of FieldCache using in-memory maps
// backed by KVStore for persistence.
type fieldCacheImpl struct {
	store kvstore.KVStore

	// In-memory caches
	fieldMappings map[string]string            // field name → field ID
	fieldOptions  map[string]map[string]string // field name → (option name → option ID)
}

// NewFieldCache creates a new FieldCache instance with empty in-memory caches.
// The cache will lazy-load data from KVStore as needed.
func NewFieldCache(store kvstore.KVStore) FieldCache {
	return &fieldCacheImpl{
		store:         store,
		fieldMappings: make(map[string]string),
		fieldOptions:  make(map[string]map[string]string),
	}
}

// GetFieldID retrieves the field ID for a given field name.
// Implements read-through caching: check cache first, then KVStore on miss.
func (c *fieldCacheImpl) GetFieldID(fieldName string) (string, error) {
	// Check cache first
	if fieldID, exists := c.fieldMappings[fieldName]; exists {
		return fieldID, nil
	}

	// Cache miss - fetch from KVStore
	fieldID, err := c.store.GetFieldMapping(fieldName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get field mapping from KVStore")
	}

	// Cache the result (even if empty string - avoids repeated KVStore lookups)
	c.fieldMappings[fieldName] = fieldID

	return fieldID, nil
}

// GetOptionID retrieves the option ID for a given field and option name.
// Implements read-through caching: check cache first, then KVStore on miss.
func (c *fieldCacheImpl) GetOptionID(fieldName, optionName string) (string, error) {
	// Check if field options are cached
	options, exists := c.fieldOptions[fieldName]
	if !exists {
		// Cache miss - fetch all options for this field from KVStore
		var err error
		options, err = c.store.GetFieldOptions(fieldName)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get field options from KVStore")
		}

		// Cache the result (even if empty - avoids repeated KVStore lookups)
		c.fieldOptions[fieldName] = options
	}

	// Look up the specific option ID
	optionID := options[optionName]
	return optionID, nil
}

// SaveFieldMapping saves a field name → ID mapping to both in-memory cache and KVStore.
// Write-through: Updates cache first (fast), then persists to KVStore.
func (c *fieldCacheImpl) SaveFieldMapping(fieldName, fieldID string) error {
	// Update in-memory cache first
	c.fieldMappings[fieldName] = fieldID

	// Persist to KVStore (write-through)
	if err := c.store.SaveFieldMapping(fieldName, fieldID); err != nil {
		return errors.Wrapf(err, "failed to save field mapping to KVStore")
	}

	return nil
}

// SaveFieldOptions saves option mappings for a field to both in-memory cache and KVStore.
// Write-through: Updates cache first (fast), then persists to KVStore.
func (c *fieldCacheImpl) SaveFieldOptions(fieldName string, options map[string]string) error {
	// Deep copy options to prevent external modifications
	optionsCopy := make(map[string]string, len(options))
	for name, id := range options {
		optionsCopy[name] = id
	}

	// Update in-memory cache first
	c.fieldOptions[fieldName] = optionsCopy

	// Persist to KVStore (write-through)
	if err := c.store.SaveFieldOptions(fieldName, options); err != nil {
		return errors.Wrapf(err, "failed to save field options to KVStore")
	}

	return nil
}

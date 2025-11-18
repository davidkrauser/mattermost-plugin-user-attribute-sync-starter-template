package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockKVStore is a mock implementation of kvstore.KVStore for testing
type MockKVStore struct {
	mock.Mock
}

func (m *MockKVStore) SaveFieldMapping(fieldName, fieldID string) error {
	args := m.Called(fieldName, fieldID)
	return args.Error(0)
}

func (m *MockKVStore) GetFieldMapping(fieldName string) (string, error) {
	args := m.Called(fieldName)
	return args.String(0), args.Error(1)
}

func (m *MockKVStore) SaveFieldOptions(fieldName string, options map[string]string) error {
	args := m.Called(fieldName, options)
	return args.Error(0)
}

func (m *MockKVStore) GetFieldOptions(fieldName string) (map[string]string, error) {
	args := m.Called(fieldName)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(map[string]string), args.Error(1)
}

func (m *MockKVStore) SaveLastSyncTime(t time.Time) error {
	args := m.Called(t)
	return args.Error(0)
}

func (m *MockKVStore) GetLastSyncTime() (time.Time, error) {
	args := m.Called()
	return args.Get(0).(time.Time), args.Error(1)
}

func TestNewFieldCache(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	assert.NotNil(t, cache)
	impl, ok := cache.(*fieldCacheImpl)
	require.True(t, ok)
	assert.NotNil(t, impl.fieldMappings)
	assert.NotNil(t, impl.fieldOptions)
	assert.Equal(t, 0, len(impl.fieldMappings))
	assert.Equal(t, 0, len(impl.fieldOptions))
}

func TestGetFieldID_CacheHit(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store).(*fieldCacheImpl)

	// Pre-populate cache
	cache.fieldMappings["department"] = "field123"

	// Should return cached value without hitting KVStore
	fieldID, err := cache.GetFieldID("department")

	assert.NoError(t, err)
	assert.Equal(t, "field123", fieldID)
	store.AssertNotCalled(t, "GetFieldMapping")
}

func TestGetFieldID_CacheMiss_Success(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// Mock KVStore to return field ID (only once)
	store.On("GetFieldMapping", "department").Return("field456", nil).Once()

	// First call - cache miss, should fetch from KVStore
	fieldID, err := cache.GetFieldID("department")

	assert.NoError(t, err)
	assert.Equal(t, "field456", fieldID)

	// Second call - cache hit, should not fetch from KVStore again
	fieldID2, err := cache.GetFieldID("department")

	assert.NoError(t, err)
	assert.Equal(t, "field456", fieldID2)

	// Verify GetFieldMapping was called exactly once
	store.AssertExpectations(t)
	store.AssertNumberOfCalls(t, "GetFieldMapping", 1)
}

func TestGetFieldID_CacheMiss_NotFound(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// Mock KVStore to return empty string (field doesn't exist)
	store.On("GetFieldMapping", "unknown_field").Return("", nil)

	fieldID, err := cache.GetFieldID("unknown_field")

	assert.NoError(t, err)
	assert.Equal(t, "", fieldID)
	store.AssertExpectations(t)
}

func TestGetFieldID_CacheMiss_Error(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// Mock KVStore to return error
	store.On("GetFieldMapping", "department").Return("", assert.AnError)

	fieldID, err := cache.GetFieldID("department")

	assert.Error(t, err)
	assert.Equal(t, "", fieldID)
	assert.Contains(t, err.Error(), "failed to get field mapping from KVStore")
	store.AssertExpectations(t)
}

func TestGetOptionID_CacheHit(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store).(*fieldCacheImpl)

	// Pre-populate cache
	cache.fieldOptions["programs"] = map[string]string{
		"Alpha": "opt123",
		"Beta":  "opt456",
	}

	// Should return cached value without hitting KVStore
	optionID, err := cache.GetOptionID("programs", "Alpha")

	assert.NoError(t, err)
	assert.Equal(t, "opt123", optionID)
	store.AssertNotCalled(t, "GetFieldOptions")
}

func TestGetOptionID_CacheMiss_Success(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// Mock KVStore to return options (only once)
	options := map[string]string{
		"Alpha": "opt789",
		"Beta":  "opt012",
	}
	store.On("GetFieldOptions", "programs").Return(options, nil).Once()

	// First call - cache miss, should fetch from KVStore
	optionID, err := cache.GetOptionID("programs", "Alpha")

	assert.NoError(t, err)
	assert.Equal(t, "opt789", optionID)

	// Second call for different option of same field - cache hit
	optionID2, err := cache.GetOptionID("programs", "Beta")

	assert.NoError(t, err)
	assert.Equal(t, "opt012", optionID2)

	// Verify GetFieldOptions was called exactly once
	store.AssertExpectations(t)
	store.AssertNumberOfCalls(t, "GetFieldOptions", 1)
}

func TestGetOptionID_OptionNotFound(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// Mock KVStore to return options (but not the one we're looking for)
	options := map[string]string{
		"Alpha": "opt789",
	}
	store.On("GetFieldOptions", "programs").Return(options, nil)

	optionID, err := cache.GetOptionID("programs", "Unknown")

	assert.NoError(t, err)
	assert.Equal(t, "", optionID) // Option doesn't exist
	store.AssertExpectations(t)
}

func TestGetOptionID_FieldNotFound(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// Mock KVStore to return empty options (field has no options)
	store.On("GetFieldOptions", "unknown_field").Return(map[string]string{}, nil)

	optionID, err := cache.GetOptionID("unknown_field", "Alpha")

	assert.NoError(t, err)
	assert.Equal(t, "", optionID)
	store.AssertExpectations(t)
}

func TestGetOptionID_CacheMiss_Error(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// Mock KVStore to return error
	store.On("GetFieldOptions", "programs").Return(map[string]string{}, assert.AnError)

	optionID, err := cache.GetOptionID("programs", "Alpha")

	assert.Error(t, err)
	assert.Equal(t, "", optionID)
	assert.Contains(t, err.Error(), "failed to get field options from KVStore")
	store.AssertExpectations(t)
}

func TestSaveFieldMapping_Success(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// Mock KVStore save
	store.On("SaveFieldMapping", "department", "field999").Return(nil)

	err := cache.SaveFieldMapping("department", "field999")

	assert.NoError(t, err)
	store.AssertExpectations(t)

	// Verify cache was updated
	fieldID, err := cache.GetFieldID("department")
	assert.NoError(t, err)
	assert.Equal(t, "field999", fieldID)
	store.AssertNotCalled(t, "GetFieldMapping") // Should hit cache, not KVStore
}

func TestSaveFieldMapping_Error(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// Mock KVStore to return error
	store.On("SaveFieldMapping", "department", "field999").Return(assert.AnError)

	err := cache.SaveFieldMapping("department", "field999")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save field mapping to KVStore")
	store.AssertExpectations(t)
}

func TestSaveFieldOptions_Success(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	options := map[string]string{
		"Alpha": "opt111",
		"Beta":  "opt222",
	}

	// Mock KVStore save
	store.On("SaveFieldOptions", "programs", options).Return(nil)

	err := cache.SaveFieldOptions("programs", options)

	assert.NoError(t, err)
	store.AssertExpectations(t)

	// Verify cache was updated
	optionID, err := cache.GetOptionID("programs", "Alpha")
	assert.NoError(t, err)
	assert.Equal(t, "opt111", optionID)
	store.AssertNotCalled(t, "GetFieldOptions") // Should hit cache, not KVStore
}

func TestSaveFieldOptions_DeepCopy(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store).(*fieldCacheImpl)

	options := map[string]string{
		"Alpha": "opt111",
	}

	store.On("SaveFieldOptions", "programs", options).Return(nil)

	err := cache.SaveFieldOptions("programs", options)
	assert.NoError(t, err)

	// Modify original map
	options["Beta"] = "opt222"

	// Verify cache was not affected (deep copy worked)
	cached := cache.fieldOptions["programs"]
	assert.Equal(t, 1, len(cached))
	assert.Equal(t, "opt111", cached["Alpha"])
	_, betaExists := cached["Beta"]
	assert.False(t, betaExists)
}

func TestSaveFieldOptions_Error(t *testing.T) {
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	options := map[string]string{
		"Alpha": "opt111",
	}

	// Mock KVStore to return error
	store.On("SaveFieldOptions", "programs", options).Return(assert.AnError)

	err := cache.SaveFieldOptions("programs", options)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save field options to KVStore")
	store.AssertExpectations(t)
}

func TestFieldCache_Integration(t *testing.T) {
	// Simulate a typical sync flow
	store := &MockKVStore{}
	cache := NewFieldCache(store)

	// 1. Field sync phase - save mappings
	store.On("SaveFieldMapping", "department", "field1").Return(nil)
	store.On("SaveFieldMapping", "location", "field2").Return(nil)
	store.On("SaveFieldOptions", "programs", map[string]string{
		"Alpha": "opt1",
		"Beta":  "opt2",
	}).Return(nil)

	err := cache.SaveFieldMapping("department", "field1")
	require.NoError(t, err)

	err = cache.SaveFieldMapping("location", "field2")
	require.NoError(t, err)

	err = cache.SaveFieldOptions("programs", map[string]string{
		"Alpha": "opt1",
		"Beta":  "opt2",
	})
	require.NoError(t, err)

	// 2. Value sync phase - read from cache (no KVStore calls)
	fieldID, err := cache.GetFieldID("department")
	assert.NoError(t, err)
	assert.Equal(t, "field1", fieldID)

	optionID, err := cache.GetOptionID("programs", "Alpha")
	assert.NoError(t, err)
	assert.Equal(t, "opt1", optionID)

	// All reads hit cache - no additional KVStore calls
	store.AssertExpectations(t)
	store.AssertNotCalled(t, "GetFieldMapping")
	store.AssertNotCalled(t, "GetFieldOptions")
}

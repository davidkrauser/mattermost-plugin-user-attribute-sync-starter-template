# Implementation Progress

This document tracks the implementation progress of the User Attribute Sync Starter Template plugin.

## Implementation Approach

**Each sub-phase is implemented as a separate commit** with this pattern:
1. **Code changes** - Production code modifications
2. **Unit tests** - Tests for those changes (if applicable)
3. **Verification** - Run `make test` and `make check-style`
4. **Commit** - Single commit with detailed explanation of WHY the changes were made, including Claude attribution

**Important:** Commit messages should NOT reference the specification or progress documents. These documents may change or be removed in the future. Instead, commit messages should:
- Explain the problem being solved
- Explain why this approach was chosen
- Describe key design decisions made in the implementation
- Stand alone as documentation of the change

After completing all sub-phases in a phase:
5. **Update documentation** - Update SPECIFICATION.md and PROGRESS.md with implementation details
6. **Add commit hashes** - Record commit hashes in PROGRESS.md for each completed sub-phase
7. **Commit documentation** - Single commit documenting the completed phase

## Phase Overview

- **Phase 1**: Foundation and Infrastructure (4 phases)
- **Phase 2**: Data Source Abstraction (3 phases)
- **Phase 3**: Field Management (7 phases)
- **Phase 4**: Value Synchronization (8 phases) - includes new Phase 4.0 for FieldCache
- **Phase 5**: Testing and Validation (1 phase)
- **Phase 6**: Documentation and Polish (3 phases)

**Total: 23 phases**

---

## Phase 1: Foundation and Infrastructure

### 1.1 - Update Plugin Metadata
**Status:** Complete
**Commit:** `91f7d81`

**Code Changes (~15-20 lines):**
- Update `plugin.json`: Change plugin ID from `com.mattermost.plugin-starter-template` to `com.mattermost.user-attribute-sync-starter-template`
- Update name, description, display name to reflect user attribute sync purpose
- Update package declarations in `server/*.go` files if needed

**Unit Tests:**
- No tests needed (metadata validation)

**Verification:**
- `make check-style` - linting passes
- Visual inspection of plugin.json

**Commit Message Guidance:**
- Explain WHY we're renaming from generic starter template to specific user attribute sync template
- Note this establishes the plugin's identity

---

### 1.2 - KVStore Keys and Helpers
**Status:** Complete
**Commit:** `5e2d6cf`

**Code Changes (~165 lines - actual):**
- Created `server/store/kvstore/sync.go`
- Defined key constants:
  - `fieldMappingPrefix`
  - `fieldOptionsPrefix`
  - `lastSyncTimestampKey`
- Implemented helpers:
  - `SaveFieldMapping(fieldName, fieldID string) error`
  - `GetFieldMapping(fieldName string) (string, error)`
  - `SaveFieldOptions(fieldName string, options map[string]string) error`
  - `GetFieldOptions(fieldName string) (map[string]string, error)`
  - `SaveLastSyncTime(t time.Time) error`
  - `GetLastSyncTime() (time.Time, error)`

**Design Decision:**
- Field options use type-safe `map[string]string` interface instead of raw JSON strings
- JSON marshaling/unmarshaling handled internally by KVStore layer
- Provides better type safety and cleaner API for callers

**Unit Tests:**
- Skipped (simple setters/getters wrapping KVStore operations)

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY we need persistent state (survives restarts, enables incremental sync)
- Reference spec sections 4.4 (state persistence) and FR8

---

### 1.3 - Property Group Helper
**Status:** Complete
**Commit:** `a30be55`

**Code Changes (~50 lines - actual):**
- Created `server/sync/property_group.go`
- Implemented `getOrRegisterCPAGroup(*pluginapi.Client) (string, error)`
- Returns Custom Profile Attributes group ID
- Handles both retrieval and registration of CPA group

**Dependency Update:**
- Updated `github.com/mattermost/mattermost/server/public` to v0.1.21
- This version includes PropertyService API with GetPropertyGroup and RegisterPropertyGroup

**Unit Tests:**
- Skipped (will be used by other sync components in later phases)
- Function currently shows as unused in linter (expected)

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY we need this helper (all CPA operations require group ID)
- Reference Mattermost PropertyService API patterns from spec Appendix B.1

---

### 1.4 - Cluster Job Setup
**Status:** Complete
**Commit:** `269b3c2`

**Code Changes (~70 lines - actual):**
- Modified `server/plugin.go` OnActivate/OnDeactivate
- Modified `server/job.go`:
  - **Hardcoded sync interval: `const syncIntervalMinutes = 60`**
  - Implemented `nextWaitInterval(now time.Time, metadata cluster.JobMetadata) time.Duration`
    - Uses `metadata.LastFinished` (actual API field name)
    - First run executes immediately (when LastFinished is zero)
    - Subsequent runs calculate wait time from last completion
  - Implemented stub `runSync()` that logs "Sync starting"
  - Set up cluster job in OnActivate using `cluster.Schedule()` with job name "AttributeSync"
  - Clean up job in OnDeactivate
- Job reference stored in Plugin struct (`backgroundJob`)

**Unit Tests:**
- Skipped (integration of existing cluster job functionality)

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY cluster jobs over goroutines (cluster-aware, leader election, failover)
- Reference spec section 4.4 (Cluster Job Lifecycle) and Appendix C.1
- Mention this prevents duplicate work in multi-server deployments
- Note hardcoded interval keeps template simple (developers can adjust as needed)

---

### Phase 1 Summary

**Status:** ✅ Complete

**Total Commits:** 4
- Phase 1.1: Update plugin metadata
- Phase 1.2: Add KVStore helpers for state management
- Phase 1.3: Add Property Group helper with dependency update
- Phase 1.4: Implement cluster-aware job scheduling

**Key Design Decisions:**
1. **Type-safe KVStore interface** - Field options use `map[string]string` with internal JSON marshaling instead of exposing raw JSON strings to callers
2. **Hardcoded sync interval** - 60 minutes, kept simple for template; developers can modify constant directly
3. **Immediate first run** - Job executes immediately on activation for quick feedback
4. **No unit tests for simple wrappers** - KVStore helpers and property group function are thin wrappers; will be tested through integration

**Dependencies Updated:**
- `github.com/mattermost/mattermost/server/public` → v0.1.21 (for PropertyService APIs)

**Ready for Phase 2:** Data Source Abstraction

---

## Phase 2: Data Source Abstraction

### 2.1 - AttributeProvider Interface
**Status:** Complete
**Commit:** `ef4f389`

**Code Changes (~37 lines - actual):**
- Created `server/sync/provider.go`
- Defined `AttributeProvider` interface with two methods:
  - `GetUserAttributes() ([]map[string]interface{}, error)`
  - `Close() error`
- Added comprehensive godoc comments explaining:
  - Stateless design from caller's perspective
  - Provider tracks internal state for incremental sync
  - Expected return format with example
  - Empty array return when no changes detected

**Unit Tests:**
- No tests needed (interface definition)

**Verification:**
- `make check-style`

---

### 2.2 - File Provider Implementation
**Status:** Complete
**Commit:** `e0bde9f`

**Code Changes (~73 lines - actual):**
- Created `server/sync/file_provider.go`
- **Changed file path to: `const defaultDataFilePath = "data/user_attributes.json"`**
  - Used `data/` directory instead of `assets/` (more semantically accurate)
- Defined `FileProvider` struct with fields:
  - `filePath string`
  - `lastReadTime time.Time`
  - `lastModTime time.Time`
- Implemented `NewFileProvider() *FileProvider` (no params, uses const)
- Implemented `GetUserAttributes()`:
  - Uses `os.Stat()` to get file modification time
  - If file mod time <= lastModTime, returns empty array (no changes)
  - Reads and parses JSON file
  - Updates `lastReadTime` and `lastModTime`
  - Returns parsed user objects
- Implemented `Close() error` (no-op)

**Unit Tests (~180 lines - actual):**
- Created `server/sync/file_provider_test.go`
- Test coverage:
  - First sync returns all users
  - Subsequent sync with unchanged file returns empty array
  - Subsequent sync after file modification returns users
  - File not found error handling
  - Invalid JSON error handling
  - Empty file handling
  - Close() method
  - Constructor default values
- Helper function `writeJSONFile()` creates temp directories and files
  - Returns both file path and directory path for test convenience

**Verification:**
- `make test` - all tests pass (28 tests)
- `make check-style` - passes

---

### 2.3 - Example Data File
**Status:** Complete
**Commit:** `be58c2b`

**Code Changes (~23 lines - actual):**
- Created `data/user_attributes.json` with 3 example users
- Includes all field types (text, multiselect, date):
  ```json
  [
    {
      "email": "john.doe@example.com",
      "department": "Engineering",
      "location": "US-East",
      "programs": ["Apples", "Oranges"],
      "start_date": "2023-01-15"
    },
    {
      "email": "jane.smith@example.com",
      "department": "Sales",
      "location": "US-West",
      "programs": ["Lemons"],
      "start_date": "2022-08-01"
    },
    {
      "email": "bob.wilson@example.com",
      "department": "Engineering",
      "location": "EU-Central",
      "programs": ["Apples", "Lemons"],
      "start_date": "2024-03-20"
    }
  ]
  ```

**Unit Tests:**
- None (data file)

**Verification:**
- JSON validated with `jq . data/user_attributes.json`

---

### Phase 2 Summary

**Status:** ✅ Complete

**Total Commits:** 3
- Phase 2.1: Add AttributeProvider interface
- Phase 2.2: Implement FileProvider with incremental sync
- Phase 2.3: Add example data file

**Key Design Decisions:**
1. **Changed directory name** - Used `data/` instead of `assets/` for semantic clarity (data can change vs static assets)
2. **os.Stat()-based incremental sync** - No separate metadata file needed; tracks modification time directly
3. **Stateless abstraction** - Provider implementations manage their own state internally
4. **Helper function returns both paths** - Test helper returns file path and directory path for test convenience

**Ready for Phase 3:** Field Management

---

## Phase 3: Field Management

### 3.1 - Type Inference
**Status:** Complete
**Commit:** `e23175a`

**Code Changes (~50 lines):**
- Create `server/sync/types.go`
- Implement `inferFieldType(value interface{}) model.PropertyFieldType`:
  - Check if array → multiselect
  - Check if string matches date pattern (YYYY-MM-DD) → date
  - Otherwise → text
- Implement date pattern regex: `^\d{4}-\d{2}-\d{2}$`

**Unit Tests (~80 lines):**
- Test array detection
- Test date detection (valid format)
- Test text fallback
- Test edge cases (nil, empty string, invalid dates)

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY automatic type inference (no manual schema definition needed)
- Reference spec FR2 and section 4.4 (step 2)
- Mention simple rules make behavior predictable

---

### 3.2 - Field Name Transformation
**Status:** Complete
**Commit:** `bed37ac`

**Code Changes (~20 lines):**
- Add to `server/sync/types.go`
- Implement `transformFieldName(name string) string`:
  - Convert snake_case/kebab-case to Title Case
  - Example: "security_clearance" → "Security Clearance"

**Unit Tests (~40 lines):**
- Test various transformations
- Test edge cases (already title case, single word, special chars)

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY transformation (user-friendly display names)
- Reference spec FR1
- Mention keeps internal names clean, display names readable

---

### 3.3 - Field Discovery
**Status:** Complete
**Commit:** `6404aa0`

**Code Changes (~40 lines):**
- Create `server/sync/field_discovery.go`
- Implement `discoverFields(users []map[string]interface{}) map[string]interface{}`:
  - Extract all unique field names (except "email")
  - Build map of field name → sample value for type inference
  - Return map

**Unit Tests (~60 lines):**
- Test with multiple users
- Test "email" exclusion
- Test with varying fields across users
- Test empty users array

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY dynamic discovery (no predefined schema)
- Reference spec FR1
- Mention this enables automatic field creation from data structure

---

### 3.4 - PropertyField Creation
**Status:** Complete
**Commit:** `dba263a`

**Code Changes (~80 lines):**
- Create `server/sync/field_sync.go`
- Implement `createPropertyField(api *pluginapi.Client, groupID, fieldName string, fieldType model.PropertyFieldType) (*model.PropertyField, error)`:
  - Build PropertyField struct
  - Set display name using transformation
  - Set CPA visibility attributes
  - Call CreatePropertyField API
  - Save mapping to KVStore
  - Return created field

**Unit Tests (~90 lines):**
- Test field creation for each type
- Test KVStore save
- Test API error handling
- Mock Property API

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY abstracted creation helper (reusable, testable)
- Reference spec section 4.4 (step 3)
- Mention proper attribute setup for CPA requirements

---

### 3.5 - Option Extraction
**Status:** Complete
**Commit:** `b268105`

**Code Changes (~30 lines):**
- Add to `server/sync/field_sync.go`
- Implement `extractMultiselectOptions(users []map[string]interface{}, fieldName string) []string`:
  - Collect all unique values for the field across users
  - Handle arrays
  - Return deduplicated list

**Unit Tests (~50 lines):**
- Test with multiple users
- Test deduplication
- Test empty arrays
- Test missing fields

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY extraction is separate (used for both create and update)
- Reference spec FR4
- Mention deduplication ensures clean option lists

---

### 3.6 - Option Merging
**Status:** Complete
**Commit:** `331934c`

**Code Changes (~60 lines):**
- Add to `server/sync/field_sync.go`
- Implement `mergeOptions(existingOptions []map[string]interface{}, newValues []string) ([]map[string]interface{}, int)`:
  - Build map of existing option name → ID
  - For each new value:
    - If exists, reuse ID
    - If new, generate new ID
  - Return merged options list and count of new options

**Unit Tests (~80 lines):**
- Test ID preservation
- Test new option addition
- Test no changes when all options exist
- Test empty existing options

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY ID preservation is critical (prevents orphaning user values)
- Reference spec FR4, Appendix C.2
- Mention append-only strategy avoids data loss

---

### 3.7 - Field Synchronization Orchestrator
**Status:** Complete
**Commit:** `1184466`

**Code Changes (~90 lines):**
- Add to `server/sync/field_sync.go`
- Implement `syncFields(api *pluginapi.Client, groupID string, users []map[string]interface{}, store *kvstore.Client) (map[string]string, error)`:
  - Discover fields
  - For each field:
    - Check if exists in KVStore
    - If not, create field
    - If multiselect, handle options
  - Return field name → ID mapping

**Unit Tests (~100 lines):**
- Test full flow with new fields
- Test with existing fields
- Test multiselect option updates
- Test partial failures
- Mock all dependencies

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY orchestrator pattern (separates concerns, testable)
- Reference spec section 4.4 (step 3)
- Mention graceful handling of partial failures

---

### Phase 3 Summary

**Status:** ✅ Complete

**Total Commits:** 7
- Phase 3.1: Add automatic field type inference from JSON data structure
- Phase 3.2: Add field name to display name transformation
- Phase 3.3: Add dynamic field discovery from user data structure
- Phase 3.4: Add Custom Profile Attribute field creation
- Phase 3.5: Add multiselect option extraction from user data
- Phase 3.6: Add append-only multiselect option merging
- Phase 3.7: Implement field synchronization orchestrator for dynamic CPA field management

**Key Design Decisions:**
1. **Enhanced date regex** - Validates month (01-12) and day (01-31) ranges for better accuracy
2. **toDisplayName() function** - Clear naming convention for human-readable field name transformation
3. **Nil value handling** - Skip nil values during field discovery rather than error
4. **Verified Mattermost CPA attributes** - Used correct constants from Mattermost codebase (`CustomProfileAttributesPropertyAttrsVisibility`, `CustomProfileAttributesPropertyAttrsManaged`)
5. **Set data structure for options** - Used `optionsSet` instead of map for semantic clarity
6. **ID generation optimization** - Compute option IDs once and reuse to avoid unnecessary calls to `model.NewId()`
7. **Append-only option merging** - Never remove options to prevent orphaning user values
8. **Graceful degradation** - Individual field failures don't block entire sync
9. **KVStore caching** - Field mappings cached to avoid redundant API calls

**Implementation Highlights:**
- Dynamic schema discovery without configuration
- Automatic type inference (text, date, multiselect)
- Complete test coverage: 150 tests passing
- Proper CPA attributes: hidden visibility, admin-managed
- Option ID stability for multiselect fields

**Ready for Phase 4:** Value Synchronization

**Note:** Phase 3.7's `syncFields()` function will be refactored in Phase 4.0 to use `FieldCache` instead of directly accessing `*kvstore.Client`. This ensures all code uses the performance-optimized cache consistently.

---

## Phase 4: Value Synchronization

### 4.0 - Field Cache Implementation
**Status:** Complete
**Commit:** `4950e5c`
**Refactoring Commit:** `11474d6` (updated syncFields to use FieldCache)

**Code Changes (~173 lines - actual):**
- Created `server/sync/field_cache.go`
- Defined `FieldCache` interface:
  - `GetFieldID(fieldName string) (string, error)` - lazy-load field mapping
  - `GetOptionID(fieldName, optionName string) (string, error)` - lazy-load option mapping
  - `SaveFieldMapping(fieldName, fieldID string) error` - write-through to cache and KVStore
  - `SaveFieldOptions(fieldName string, options map[string]string) error` - write-through
  - **Note:** Removed `Load()` method - cache uses lazy-loading instead of eager-loading
- Implemented `fieldCacheImpl` struct:
  - `store kvstore.KVStore` - backing storage
  - `fieldMappings map[string]string` - in-memory cache for field name → ID
  - `fieldOptions map[string]map[string]string` - in-memory cache for option mappings
- Implemented constructor `NewFieldCache(store kvstore.KVStore) FieldCache`
- Implemented lazy-loading read-through caching strategy
- Implemented write-through caching (updates both memory and KVStore)

**Design Decision - Lazy Loading:**
Changed from eager-loading to lazy-loading because KVStore doesn't support "list all keys" operation. With lazy-loading:
- First lookup: Fetches from KVStore and caches result
- Subsequent lookups: Returns cached value instantly
- Result: Each unique field/option loaded exactly once per sync

**Unit Tests (~357 lines - actual):**
- Test GetFieldID() with cache hit and cache miss
- Test GetFieldID() caches empty results (prevents repeated lookups)
- Test GetOptionID() with nested lookups
- Test GetOptionID() caches entire field's options on first access
- Test SaveFieldMapping() updates both cache and KVStore
- Test SaveFieldOptions() updates both cache and KVStore
- Test deep copy in SaveFieldOptions() prevents external modifications
- Test error handling for KVStore failures
- Test integration scenario (field sync → value sync with cache)
- Mock KVStore interface

**Verification:**
- ✅ `make test` - All 166 tests pass
- ✅ `make check-style` - Passes

**Refactoring Impact (commit 11474d6):**
Updated all field sync code to use FieldCache instead of KVStore directly:
- Modified `syncFields()` signature to accept `FieldCache` instead of `kvstore.KVStore`
- Modified `createPropertyField()` to use FieldCache for saving mappings
- Modified `createMultiselectFieldWithOptions()` to use FieldCache
- Modified `updateMultiselectOptions()` to use FieldCache
- Updated all test mocks from `mockKVStore` to `mockFieldCache`
- Removed kvstore import from field_sync.go

---

### 4.1 - User Resolution
**Status:** Not Started

**Code Changes (~25 lines):**
- Create `server/sync/value_sync.go`
- Implement `resolveUserByEmail(api *pluginapi.Client, email string) (*model.User, error)`:
  - Call GetUserByEmail
  - Return user or error

**Unit Tests (~40 lines):**
- Test successful resolution
- Test user not found
- Test API error
- Mock API

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY email-based resolution (most reliable cross-system identifier)
- Reference spec FR5
- Mention errors are expected and handled gracefully by caller

---

### 4.2 - Value Formatting - Text and Date
**Status:** Not Started

**Code Changes (~30 lines):**
- Add to `server/sync/value_sync.go`
- Implement `formatTextValue(value interface{}) (json.RawMessage, error)`:
  - Marshal string value to JSON
- Implement `formatDateValue(value interface{}) (json.RawMessage, error)`:
  - Validate date format
  - Marshal to JSON

**Unit Tests (~50 lines):**
- Test text formatting
- Test date formatting
- Test invalid inputs

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY type-specific formatters (PropertyService requires JSON encoding)
- Reference spec Appendix B.3
- Mention validation catches bad data early

---

### 4.3 - Value Formatting - Multiselect
**Status:** Not Started

**Code Changes (~50 lines):**
- Add to `server/sync/value_sync.go`
- Implement `formatMultiselectValue(values interface{}, fieldName string, cache FieldCache) (json.RawMessage, error)`:
  - Get option name → ID mapping from FieldCache (in-memory lookup)
  - Convert array of names to array of IDs
  - Marshal to JSON
  - Handle missing options

**Unit Tests (~70 lines):**
- Test option ID lookup from cache
- Test multiple values
- Test empty array
- Test missing option error
- Mock FieldCache

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY option ID conversion (Mattermost stores IDs not names)
- Explain WHY FieldCache instead of PropertyField (avoids API calls, uses cached data)
- Reference spec section 4.4 (step 4)
- Mention validation prevents invalid option references

---

### 4.4 - PropertyValue Construction
**Status:** Not Started

**Code Changes (~70 lines):**
- Add to `server/sync/value_sync.go`
- Implement `buildPropertyValues(api *pluginapi.Client, user *model.User, groupID string, userAttrs map[string]interface{}, cache FieldCache) ([]*model.PropertyValue, error)`:
  - Loop through user attributes (except email)
  - Look up field ID from cache
  - Format value based on type (text/date/multiselect)
  - Build PropertyValue structs
  - Return array

**Unit Tests (~90 lines):**
- Test all field types
- Test missing fields
- Test format errors
- Mock FieldCache

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY batch construction (prepares for bulk upsert)
- Explain WHY FieldCache simplifies function signature (no need to pass field mappings separately)
- Reference spec section 4.4 (step 4)
- Mention per-user batching for atomicity

---

### 4.5 - Batch Upsert
**Status:** Not Started

**Code Changes (~25 lines):**
- Add to `server/sync/value_sync.go`
- Implement `upsertPropertyValues(api *pluginapi.Client, values []*model.PropertyValue) error`:
  - Call UpsertPropertyValues API
  - Handle errors

**Unit Tests (~40 lines):**
- Test successful upsert
- Test empty array
- Test API error
- Mock API

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY bulk upsert (performance optimization)
- Reference spec NFR1
- Mention Mattermost API handles create vs update logic

---

### 4.6 - User Sync Orchestrator
**Status:** Not Started

**Code Changes (~70 lines):**
- Add to `server/sync/value_sync.go`
- Implement `syncUsers(api *pluginapi.Client, groupID string, users []map[string]interface{}, cache FieldCache) error`:
  - Loop through users
  - Resolve by email (skip if not found)
  - Build property values using cache
  - Upsert
  - Log errors but continue with next user

**Unit Tests (~100 lines):**
- Test successful sync
- Test user not found handling
- Test partial failures
- Mock all dependencies (API, FieldCache)

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY partial failure handling (don't fail entire sync for one user)
- Explain WHY simplified signature (FieldCache encapsulates field mappings and options)
- Reference spec FR7
- Mention graceful degradation enables progress despite individual failures
- Note: Removed statistics tracking (successCount/skippedCount) per user feedback - logging per-user is sufficient

---

### 4.7 - Main Sync Orchestrator
**Status:** Not Started

**Code Changes (~100 lines):**
- Update `server/job.go` runSync function:
  - Initialize FieldCache and load from KVStore
  - Initialize provider
  - Fetch users from provider
  - If no users, return early (log and exit)
  - Get or register CPA group
  - Sync fields (pass cache)
  - Sync values (pass cache)
  - Update last sync timestamp in KVStore
  - Log comprehensive summary with error wrapping

**Unit Tests (~120 lines):**
- Test full sync flow with cache
- Test empty users handling
- Test cache initialization errors
- Test provider errors
- Test field sync errors
- Test value sync errors
- Mock all dependencies (provider, FieldCache, API, KVStore)

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY orchestrator coordinates entire flow (single entry point, manages lifecycle)
- Explain WHY FieldCache initialization happens here (per-sync lifecycle, fresh data)
- Reference spec section 4.4 (complete sync workflow)
- Mention error wrapping provides full context when issues occur
- Note this completes the core sync implementation

---

## Phase 5: Testing and Validation

### 5.1 - Integration Test
**Status:** Not Started

**Code Changes (~0 lines, test only):**

**Unit Tests (~150 lines):**
- Create `server/integration_test.go`
- Test complete sync flow end-to-end:
  - Mock Mattermost APIs
  - Use real file provider with test data
  - Verify fields created
  - Verify values synced
  - Verify incremental sync
- Test multiple sync runs

**Verification:**
- `make test`
- All tests pass

**Commit Message Guidance:**
- Explain WHY integration test (validates all components work together)
- Reference spec Phase 5
- Mention this provides confidence in full system behavior

---

## Phase 6: Documentation and Polish

### 6.1 - Update README
**Status:** Not Started

**Code Changes (~0 lines, documentation):**
- Update `README.md`:
  - Replace generic template content
  - Add overview of user attribute sync
  - Add setup instructions
  - Add hardcoded values documentation (interval, file path)
  - Add example data format
  - Add testing instructions
  - Add customization guide

**Unit Tests:**
- None (documentation)

**Verification:**
- Visual review of README

**Commit Message Guidance:**
- Explain WHY comprehensive README (first stop for developers)
- Reference spec section 5.1
- Mention this enables developers to get started quickly

---

### 6.2 - Add Integration Guide
**Status:** Not Started

**Code Changes (~0 lines, documentation):**
- Create `docs/INTEGRATION_GUIDE.md`:
  - How to implement custom AttributeProvider
  - REST API provider example
  - LDAP provider example
  - Authentication patterns
  - Testing strategies
  - How to modify hardcoded values

**Unit Tests:**
- None (documentation)

**Verification:**
- Visual review

**Commit Message Guidance:**
- Explain WHY integration guide (core value of template)
- Reference spec section 5.2
- Mention this shows developers how to adapt template

---

### 6.3 - Code Documentation Review
**Status:** Not Started

**Code Changes (~varies, inline comments):**
- Review all public functions for godoc comments
- Add "why" comments to complex logic
- Add references to spec sections where relevant

**Unit Tests:**
- None (documentation)

**Verification:**
- `make check-style`
- Visual review

**Commit Message Guidance:**
- Explain WHY inline documentation matters (code as teaching tool)
- Reference spec NFR3
- Mention this makes template educational, not just functional

---

## Summary

**Total Phases:** 23
- Phase 1 (Foundation): 4 phases
- Phase 2 (Data Source): 3 phases
- Phase 3 (Field Management): 7 phases
- Phase 4 (Value Sync): 8 phases (includes FieldCache)
- Phase 5 (Testing): 1 phase
- Phase 6 (Documentation): 3 phases

**Key Design Decisions:**
- No configuration system - hardcoded values
- Sync interval: 60 minutes (hardcoded)
- File path: `data/user_attributes.json` (hardcoded)
- No metadata file - use `os.Stat()` for file modification tracking
- FieldCache: Per-sync in-memory cache wrapping KVStore for performance
- Each phase = one commit with code + tests + verification

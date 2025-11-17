# Implementation Progress

This document tracks the implementation progress of the User Attribute Sync Starter Template plugin.

## Implementation Approach

Each phase follows this pattern:
1. **Code changes** - Production code modifications
2. **Unit tests** - Tests for those changes
3. **Verification** - Run `make test` and `make check-style`
4. **Commit** - Single commit with detailed explanation of WHY the changes were made, including Claude attribution

## Phase Overview

- **Phase 1**: Foundation and Infrastructure (4 phases)
- **Phase 2**: Data Source Abstraction (3 phases)
- **Phase 3**: Field Management (7 phases)
- **Phase 4**: Value Synchronization (7 phases)
- **Phase 5**: Testing and Validation (1 phase)
- **Phase 6**: Documentation and Polish (3 phases)

**Total: 22 phases**

---

## Phase 1: Foundation and Infrastructure

### 1.1 - Update Plugin Metadata
**Status:** Not Started

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
**Status:** Not Started

**Code Changes (~60 lines):**
- Create `server/store/kvstore/sync.go`
- Define key constants:
  - `fieldMappingPrefix`
  - `fieldOptionsPrefix`
  - `lastSyncTimestampKey`
- Implement helpers:
  - `SaveFieldMapping(fieldName, fieldID string) error`
  - `GetFieldMapping(fieldName string) (string, error)`
  - `SaveLastSyncTime(t time.Time) error`
  - `GetLastSyncTime() (time.Time, error)`

**Unit Tests (~80 lines):**
- Mock KVStore
- Test all CRUD operations
- Test error handling

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY we need persistent state (survives restarts, enables incremental sync)
- Reference spec sections 4.4 (state persistence) and FR8

---

### 1.3 - Property Group Helper
**Status:** Not Started

**Code Changes (~30 lines):**
- Create `server/sync/property_group.go`
- Implement `getOrRegisterCPAGroup(*pluginapi.Client) (string, error)`
- Returns Custom Profile Attributes group ID

**Unit Tests (~40 lines):**
- Mock Property API
- Test successful registration
- Test error handling

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY we need this helper (all CPA operations require group ID)
- Reference Mattermost PropertyService API patterns from spec Appendix B.1

---

### 1.4 - Cluster Job Setup
**Status:** Not Started

**Code Changes (~80 lines):**
- Modify `server/plugin.go` OnActivate/OnDeactivate
- Modify `server/job.go`:
  - **Hardcode sync interval: `const syncIntervalMinutes = 60`**
  - Implement `nextWaitInterval(now time.Time, metadata cluster.JobMetadata) time.Duration`
  - Implement stub `runSync()` that just logs "Sync starting"
  - Set up cluster job in OnActivate using `cluster.Schedule()`
  - Clean up job in OnDeactivate
- Store job reference in Plugin struct

**Unit Tests (~60 lines):**
- Test nextWaitInterval logic (first run vs subsequent)
- Test interval calculation
- Mock cluster scheduler

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY cluster jobs over goroutines (cluster-aware, leader election, failover)
- Reference spec section 4.4 (Cluster Job Lifecycle) and Appendix C.1
- Mention this prevents duplicate work in multi-server deployments
- Note hardcoded interval keeps template simple (developers can adjust as needed)

---

## Phase 2: Data Source Abstraction

### 2.1 - AttributeProvider Interface
**Status:** Not Started

**Code Changes (~20 lines):**
- Create `server/sync/provider.go`
- Define `AttributeProvider` interface:
  ```go
  type AttributeProvider interface {
      GetUserAttributes() ([]map[string]interface{}, error)
      Close() error
  }
  ```
- Add comprehensive godoc comments explaining contract

**Unit Tests:**
- No tests needed (interface definition)

**Verification:**
- `make check-style`

**Commit Message Guidance:**
- Explain WHY interface abstraction (swappable data sources)
- Reference spec section 4.5 design benefits
- Mention stateless design where provider tracks internal state

---

### 2.2 - File Provider Implementation
**Status:** Not Started

**Code Changes (~90 lines):**
- Create `server/sync/file_provider.go`
- **Hardcode file path: `const defaultDataFilePath = "assets/user_attributes.json"`**
- Define `FileProvider` struct with fields:
  - `filePath string`
  - `lastReadTime time.Time`
  - `lastModTime time.Time`
- Implement `NewFileProvider() *FileProvider` (no params, uses const)
- Implement `GetUserAttributes()`:
  - Use `os.Stat()` to get file modification time
  - If file mod time <= lastModTime, return empty array (no changes)
  - Read and parse JSON file
  - Update `lastReadTime` and `lastModTime`
  - Return parsed user objects
- Implement `Close() error` (no-op)

**Unit Tests (~100 lines):**
- Test first sync returns all users
- Test subsequent sync with unchanged file returns empty array
- Test subsequent sync after file modification returns users
- Test file not found error
- Test invalid JSON error
- Mock file system operations using test files

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY file-based provider with os.Stat() (simple, no metadata file needed)
- Reference spec section 4.5.2 but note simplification
- Mention this is the reference implementation developers learn from
- Note hardcoded path keeps template simple

---

### 2.3 - Example Data File
**Status:** Not Started

**Code Changes (~0 lines, but create file):**
- Create `assets/user_attributes.json` with 3-5 example users
- Include different field types (text, multiselect, date):
  ```json
  [
    {
      "email": "john.doe@example.com",
      "department": "Engineering",
      "location": "US-East",
      "security_clearance": ["Level2", "Level3"],
      "start_date": "2023-01-15"
    },
    {
      "email": "jane.smith@example.com",
      "department": "Sales",
      "location": "US-West",
      "security_clearance": ["Level1"],
      "start_date": "2022-08-01"
    }
  ]
  ```

**Unit Tests:**
- None (data file)

**Verification:**
- Visual inspection of JSON validity
- Can manually run `jq . assets/user_attributes.json` to validate

**Commit Message Guidance:**
- Explain WHY example data is crucial (developers need working examples)
- Reference spec Appendix A
- Mention examples demonstrate all supported field types

---

## Phase 3: Field Management

### 3.1 - Type Inference
**Status:** Not Started

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
**Status:** Not Started

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
**Status:** Not Started

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
**Status:** Not Started

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
**Status:** Not Started

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
**Status:** Not Started

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
**Status:** Not Started

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

## Phase 4: Value Synchronization

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
- Implement `formatMultiselectValue(values interface{}, field *model.PropertyField) (json.RawMessage, error)`:
  - Extract option name → ID mapping from field
  - Convert array of names to array of IDs
  - Marshal to JSON
  - Handle missing options

**Unit Tests (~70 lines):**
- Test option ID lookup
- Test multiple values
- Test empty array
- Test missing option error

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY option ID conversion (Mattermost stores IDs not names)
- Reference spec section 4.4 (step 4)
- Mention validation prevents invalid option references

---

### 4.4 - PropertyValue Construction
**Status:** Not Started

**Code Changes (~70 lines):**
- Add to `server/sync/value_sync.go`
- Implement `buildPropertyValues(api *pluginapi.Client, user *model.User, groupID string, userAttrs map[string]interface{}, fieldMappings map[string]string, fields map[string]*model.PropertyField) ([]*model.PropertyValue, error)`:
  - Loop through user attributes (except email)
  - Look up field ID and field definition
  - Format value based on type
  - Build PropertyValue structs
  - Return array

**Unit Tests (~90 lines):**
- Test all field types
- Test missing fields
- Test format errors
- Mock field data

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY batch construction (prepares for bulk upsert)
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

**Code Changes (~80 lines):**
- Add to `server/sync/value_sync.go`
- Implement `syncUsers(api *pluginapi.Client, groupID string, users []map[string]interface{}, fieldMappings map[string]string, fields map[string]*model.PropertyField) (successCount, skippedCount int, err error)`:
  - Loop through users
  - Resolve by email (skip if not found)
  - Build property values
  - Upsert
  - Track statistics
  - Log errors but continue

**Unit Tests (~100 lines):**
- Test successful sync
- Test user not found handling
- Test partial failures
- Test statistics tracking
- Mock all dependencies

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY partial failure handling (don't fail entire sync for one user)
- Reference spec FR7
- Mention statistics enable observability

---

### 4.7 - Main Sync Orchestrator
**Status:** Not Started

**Code Changes (~90 lines):**
- Update `server/job.go` runSync function:
  - Initialize provider
  - Fetch users from provider
  - If no users, return early
  - Sync fields
  - Query field definitions for multiselect
  - Sync values
  - Update last sync timestamp
  - Log comprehensive summary

**Unit Tests (~120 lines):**
- Test full sync flow
- Test empty users handling
- Test provider errors
- Test field sync errors
- Test value sync errors
- Mock all dependencies

**Verification:**
- `make test`
- `make check-style`

**Commit Message Guidance:**
- Explain WHY orchestrator coordinates entire flow (single entry point)
- Reference spec section 4.4 (all steps)
- Mention this completes the sync implementation

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

**Total Phases:** 22
- Phase 1 (Foundation): 4 phases
- Phase 2 (Data Source): 3 phases
- Phase 3 (Field Management): 7 phases
- Phase 4 (Value Sync): 7 phases
- Phase 5 (Testing): 1 phase
- Phase 6 (Documentation): 3 phases

**Key Design Decisions:**
- No configuration system - hardcoded values
- Sync interval: 60 minutes (hardcoded)
- File path: `assets/user_attributes.json` (hardcoded)
- No metadata file - use `os.Stat()` for file modification tracking
- Each phase = one commit with code + tests + verification

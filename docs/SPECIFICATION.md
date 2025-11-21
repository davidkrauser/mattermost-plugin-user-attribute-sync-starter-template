# User Attribute Sync Starter Template - Specification

## 1. Overview

### 1.1 Purpose and Scope

This specification defines the **User Attribute Sync Starter Template**, a reference implementation Mattermost plugin that demonstrates how to synchronize user profile attributes from external systems into Mattermost's Custom Profile Attributes (CPA) system.

**Primary Purpose:**
Provide plugin developers with a working, well-documented example of how to:
- Fetch user attribute data from external systems
- Create Custom Profile Attribute fields with hardcoded schema definitions
- Synchronize attribute values for users in an incremental, efficient manner
- Implement cluster-aware background jobs for reliable synchronization

**Scope:**
This is a **starter template** and **educational resource**, not a production-ready integration. It demonstrates best practices and patterns that developers should adapt for their specific use cases.

**In Scope:**
- Hardcoded field schema with explicit type definitions
- Incremental user attribute value synchronization
- Cluster-aware background job scheduling
- Abstracted data source interface (swap file/API/LDAP)
- Comprehensive error handling and logging patterns
- Email-based user identification
- One-time field synchronization on plugin activation

**Out of Scope:**
- Dynamic field discovery from data structure
- Type inference (text, date, multiselect)
- Display name transformation (snake_case → Title Case)
- Field caching in KVStore
- Runtime option detection and merging
- Production-ready error recovery mechanisms
- Advanced conflict resolution strategies
- Bi-directional synchronization (Mattermost → external system)
- User provisioning/deprovisioning
- Field deletion or type changes
- UI components for monitoring (server-only plugin)
- Authentication/authorization for external systems (implementation-specific)

### 1.2 Target Audience

**Primary Audience: Plugin Developers**

This specification and template are designed for developers who need to:
- Integrate Mattermost with external identity/attribute management systems
- Synchronize user data from HR systems, LDAP directories, or custom databases
- Populate Custom Profile Attributes for use with Attribute-Based Access Control (ABAC)
- Learn Mattermost plugin development best practices

**Expected Knowledge:**
- Go programming language
- Mattermost plugin architecture basics
- REST APIs and JSON data handling
- Basic understanding of Custom Profile Attributes

**Not Required:**
- Deep knowledge of Mattermost internals
- Experience with specific external systems (LDAP, SAML, etc.)
- Frontend/React development (server-only plugin)

### 1.3 What This Template Demonstrates

This starter template showcases the following patterns and capabilities:

#### Core Functionality
1. **Hardcoded Field Schema**
   - Explicit field definitions with human-readable IDs
   - Clear field type declarations (text, multiselect, date)
   - Fixed multiselect option IDs for stable references
   - One-time field creation/update on plugin activation

2. **Incremental Value Synchronization**
   - Process only changed users after initial sync
   - Track file modification time for change detection
   - Handle partial failures gracefully
   - Efficient bulk value upserts

3. **Data Source Abstraction**
   - Interface-based provider pattern
   - Easy to swap file/API/database implementations
   - Example file-based provider included

#### Mattermost Plugin Patterns
4. **Cluster-Aware Jobs**
   - Use Mattermost cluster job system
   - Prevent duplicate work in multi-server deployments
   - Configurable sync intervals

5. **Property API Usage**
   - Create and update PropertyFields
   - Bulk upsert PropertyValues efficiently
   - Idempotent field creation (safe to restart)

6. **Error Handling**
   - Graceful degradation for partial failures
   - Comprehensive structured logging
   - Retry-safe idempotent operations

7. **State Management**
   - Use KVStore for sync timestamp (incremental sync)
   - Minimal state storage (no field caching needed)

#### Code Quality
8. **Well-Documented Code**
   - Extensive inline comments
   - Clear separation of concerns
   - Reusable, testable components

9. **Testing Strategies**
   - Unit tests for core logic
   - Mock external data sources
   - Example test data included

### 1.4 Design Philosophy

**Simplicity Over Flexibility**

This template intentionally uses a hardcoded field schema rather than dynamic field discovery. This design choice provides:

**Benefits:**
- **~80% less code** - Eliminates complex discovery, inference, and caching logic
- **Explicit and predictable** - Developers see exactly what fields are created
- **Easier to customize** - Modify the `fieldDefinitions` array directly
- **Production-appropriate** - Most real integrations know their schema upfront
- **Better for learning** - Simpler code is easier to understand and adapt

**Tradeoffs:**
- Requires code changes to add new fields (not runtime configuration)
- No automatic adaptation to external system schema changes
- Must manually define field types and option IDs

**When This Approach Works Best:**
- External system schema is known and stable
- Field additions are infrequent (quarterly or less)
- Strong preference for explicit, reviewable code changes
- Team values code clarity over runtime flexibility

**When Dynamic Discovery Might Be Better:**
- External system schema changes frequently
- Hundreds of attributes that vary by tenant
- No control over external data structure
- Need to support arbitrary custom fields

### 1.5 How to Use This Specification

**For Plugin Developers:**
1. Read sections 1-3 to understand the architecture and design
2. Review section 4 to understand the AttributeProvider interface
3. Study section 5 for implementation details
4. Use section 6 as a guide when customizing for your external system
5. Reference appendices for example data formats and API details

**For System Architects:**
- Focus on sections 2-3 for high-level design and requirements
- Review section 3.4 (Design Constraints) to understand tradeoffs
- Use this as a reference for planning custom integrations

**For Code Reviewers:**
- Section 5 provides detailed implementation specifications
- Appendices contain API references and testing strategies

---

## 2. Requirements

### 2.1 Functional Requirements

#### FR1: Hardcoded Field Creation
**Requirement:** The plugin SHALL create Custom Profile Attribute fields based on hardcoded definitions in code.

**Field Schema:**
- **job_title** (text) - User's job title
- **programs** (multiselect) - Programs with options: Apples, Oranges, Lemons
- **start_date** (date) - User's start date in YYYY-MM-DD format

**Field Attributes:**
- All fields marked as `visibility: hidden` (not shown in profile/user card)
- All fields marked as `managed: admin` (users cannot edit)

**Field IDs:**
- Fields use human-readable IDs: `field_job_title`, `field_programs`, `field_start_date`
- Multiselect options use human-readable IDs: `option_apples`, `option_oranges`, `option_lemons`

**Acceptance Criteria:**
- Fields created once during plugin activation (OnActivate)
- If fields already exist, they are updated to match the hardcoded definition
- Field creation is idempotent (safe to restart plugin)
- Field sync completes before background job starts

#### FR2: Value Synchronization
**Requirement:** The plugin SHALL synchronize user attribute values from external data source to Mattermost PropertyValues.

**Data Format:**
```json
[
  {
    "email": "user@example.com",
    "job_title": "Software Engineer",
    "programs": ["Apples", "Oranges"],
    "start_date": "2023-01-15"
  }
]
```

**Synchronization Rules:**
- Match users by email address
- Skip users not found in Mattermost
- Convert multiselect option names to option IDs using hardcoded mappings
- Format all values as JSON for PropertyService API
- Use bulk upsert for efficiency

**Acceptance Criteria:**
- All user values synced in single bulk operation per user
- Individual user failures don't block other users
- Unknown fields are skipped with warning log
- Unknown multiselect options cause error (indicates data/schema mismatch)

#### FR3: Incremental Synchronization
**Requirement:** The plugin SHALL support incremental synchronization to process only changed data.

**Implementation:**
- File-based provider tracks file modification time
- On first sync: process all users
- On subsequent syncs: process only if file modified since last sync
- If file unchanged: return empty array (no sync needed)

**Acceptance Criteria:**
- First sync processes all users from data source
- Subsequent syncs skip unchanged data
- State survives plugin restarts
- Sync timestamp stored in KVStore

#### FR4: User Resolution
**Requirement:** The plugin SHALL resolve Mattermost users by email address.

**Behavior:**
- Extract "email" field from external user data
- Use Mattermost `GetUserByEmail` API
- Skip users not found (log warning)
- Continue processing other users on failure

**Acceptance Criteria:**
- Email field is required in external data
- Case-sensitive email matching
- User not found is non-fatal (graceful degradation)

#### FR5: Background Job Scheduling
**Requirement:** The plugin SHALL use cluster-aware background jobs for periodic synchronization.

**Implementation:**
- Use Mattermost `cluster.Schedule` API
- Sync interval: 60 minutes (hardcoded)
- First run: immediate (0 wait)
- Subsequent runs: fixed interval from last completion

**Acceptance Criteria:**
- Only one job runs in multi-server cluster
- Job automatically restarts on leader failover
- Job cleaned up on plugin deactivation

### 2.2 Non-Functional Requirements

#### NFR1: Performance
**Requirement:** The plugin SHALL synchronize 1000 users in under 30 seconds.

**Implementation:**
- Bulk upsert PropertyValues (batch per user)
- Minimal API calls (one group lookup, one field sync, values in batches)
- No unnecessary field updates on each sync

#### NFR2: Reliability
**Requirement:** The plugin SHALL handle partial failures gracefully.

**Graceful Degradation:**
- Individual field creation failure → continue with other fields
- Individual user sync failure → continue with other users
- All operations logged with structured context

#### NFR3: Maintainability
**Requirement:** Code SHALL be well-documented and easy to understand.

**Documentation:**
- Inline comments explain WHY, not just WHAT
- Function-level documentation for public APIs
- Examples in comments
- Clear separation of concerns

#### NFR4: Testability
**Requirement:** Core logic SHALL be unit testable with mocked dependencies.

**Testing:**
- Interface-based design enables mocking
- Unit tests for field sync, value sync, providers
- Example test data included
- Tests verify error handling and edge cases

---

## 3. Architecture

### 3.1 High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                    Mattermost Plugin                        │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Plugin Activation (OnActivate)          │  │
│  │                                                      │  │
│  │  1. Get/Register CPA Group                          │  │
│  │  2. Sync Hardcoded Field Definitions (once)         │  │
│  │  3. Start Background Job                            │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                                 │
│                           ▼                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Background Job (Every 60 minutes)            │  │
│  │                                                      │  │
│  │  1. Fetch Changed Users (AttributeProvider)         │  │
│  │  2. Resolve Users by Email                          │  │
│  │  3. Build PropertyValues (map to field IDs)         │  │
│  │  4. Bulk Upsert Values                              │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
         │                                    │
         ▼                                    ▼
┌──────────────────┐              ┌───────────────────────┐
│  External Data   │              │ Mattermost Property   │
│  Source          │              │ Service APIs          │
│  (File/API/LDAP) │              │ - CreatePropertyField │
│                  │              │ - UpdatePropertyField │
└──────────────────┘              │ - UpsertPropertyValues│
                                  └───────────────────────┘
```

### 3.2 Component Overview

#### Field Synchronization Layer (`server/sync/field_sync.go`)
- **Purpose:** Create/update PropertyFields from hardcoded definitions
- **When:** Once during plugin activation
- **Idempotent:** Safe to run multiple times
- **Key Functions:**
  - `SyncFields()` - Iterate hardcoded definitions, create/update each
  - `createOrUpdateField()` - Idempotent field creation
  - `GetFieldID()` - Map external field name → Mattermost field ID
  - `GetProgramOptionID()` - Map option name → option ID

**Hardcoded Schema:**
```go
const (
    FieldIDJobTitle  = "field_job_title"
    FieldIDPrograms  = "field_programs"
    FieldIDStartDate = "field_start_date"

    OptionIDApples  = "option_apples"
    OptionIDOranges = "option_oranges"
    OptionIDLemons  = "option_lemons"
)

var fieldDefinitions = []fieldDefinition{
    {ID: FieldIDJobTitle, Name: "Job Title", Type: model.PropertyFieldTypeText},
    {ID: FieldIDPrograms, Name: "Programs", Type: model.PropertyFieldTypeMultiselect,
        Options: []map[string]interface{}{
            {"id": OptionIDApples, "name": "Apples"},
            {"id": OptionIDOranges, "name": "Oranges"},
            {"id": OptionIDLemons, "name": "Lemons"},
        },
    },
    {ID: FieldIDStartDate, Name: "Start Date", Type: model.PropertyFieldTypeDate},
}
```

#### Value Synchronization Layer (`server/sync/value_sync.go`)
- **Purpose:** Synchronize user attribute values
- **When:** Periodically (every 60 minutes)
- **Key Functions:**
  - `SyncUsers()` - Orchestrate value sync for all users
  - `buildPropertyValues()` - Convert external data → PropertyValues
  - `formatStringValue()` - JSON-encode text/date values
  - `formatMultiselectValue()` - Convert option names → option IDs, JSON-encode

**Value Formatting:**
- Text fields: `"Software Engineer"` → `json.RawMessage("\"Software Engineer\"")`
- Date fields: `"2023-01-15"` → `json.RawMessage("\"2023-01-15\"")`
- Multiselect: `["Apples", "Oranges"]` → `json.RawMessage("[\"option_apples\",\"option_oranges\"]")`

#### Data Provider Layer (`server/sync/provider.go`, `server/sync/file_provider.go`)
- **Purpose:** Abstract external data source access
- **Interface:** `AttributeProvider`
  - `GetUserAttributes()` - Fetch user data (incremental)
  - `Close()` - Release resources
- **Implementation:** FileProvider
  - Reads `data/user_attributes.json`
  - Tracks file modification time
  - Returns empty array if file unchanged

#### Job Orchestration (`server/job.go`)
- **Purpose:** Schedule and run periodic sync
- **Interval:** 60 minutes (hardcoded constant)
- **First Run:** Immediate
- **Workflow:**
  1. Fetch users from provider
  2. Skip if no changes
  3. Get CPA group ID
  4. Sync user values (no field sync - already done in OnActivate)

#### Plugin Activation (`server/plugin.go`)
- **Purpose:** Initialize plugin and sync fields once
- **OnActivate:**
  1. Initialize API clients
  2. Get/Register CPA group
  3. **Sync hardcoded field definitions** ← One-time field sync
  4. Start background job for value sync

### 3.3 Data Flow

#### Plugin Activation Flow (One-Time)
```
OnActivate()
   │
   ├─> Get/Register CPA Group
   │
   ├─> SyncFields(client, groupID)
   │      │
   │      ├─> For each hardcoded field definition
   │      │      │
   │      │      ├─> CreatePropertyField(field)
   │      │      │      │
   │      │      │      ├─> Success → Done
   │      │      │      │
   │      │      │      └─> Failure (already exists) → GetPropertyField()
   │      │      │                                          │
   │      │      │                                          └─> UpdatePropertyField()
   │      │      │
   │      │      └─> Next field
   │      │
   │      └─> Return (all fields processed)
   │
   └─> Start Background Job
```

#### Periodic Sync Flow (Value Sync Only)
```
runSync() [every 60 min]
   │
   ├─> provider.GetUserAttributes()
   │      │
   │      └─> Check file modification time
   │             │
   │             ├─> Modified → Read & parse JSON
   │             └─> Unchanged → Return []
   │
   ├─> If no users → Exit
   │
   ├─> Get CPA Group ID
   │
   └─> SyncUsers(client, groupID, users)
          │
          └─> For each user
                 │
                 ├─> GetUserByEmail(user.email)
                 │      │
                 │      ├─> Success → Continue
                 │      └─> Not Found → Skip (log warning)
                 │
                 ├─> buildPropertyValues(user, attributes)
                 │      │
                 │      ├─> For each attribute
                 │      │      │
                 │      │      ├─> GetFieldID(fieldName) → field ID
                 │      │      │
                 │      │      ├─> Format value by type:
                 │      │      │      │
                 │      │      │      ├─> []string → formatMultiselectValue()
                 │      │      │      │                  │
                 │      │      │      │                  └─> GetProgramOptionID(name) → option ID
                 │      │      │      │
                 │      │      │      └─> string → formatStringValue()
                 │      │      │
                 │      │      └─> Build PropertyValue object
                 │      │
                 │      └─> Return []*PropertyValue
                 │
                 └─> UpsertPropertyValues(values)
```

### 3.4 Design Constraints and Decisions

#### 3.4.1 Hardcoded Fields (No Dynamic Discovery)

**Decision:** Fields are explicitly defined in code with human-readable IDs.

**Rationale:**
- **Simplicity:** 80% less code than dynamic discovery approach
- **Clarity:** Developers see exactly what fields are created
- **Stability:** No surprises from data structure changes
- **Maintainability:** Easier to test, review, and debug
- **Production Pattern:** Most integrations know their schema upfront

**Implications:**
- Adding new fields requires code changes (not runtime config)
- Field types must be specified explicitly
- Multiselect options must be predefined
- No automatic adaptation to external schema changes

**How to Add Fields:**
1. Add constants to `field_sync.go` (field ID, option IDs)
2. Add mapping to `fieldNameToID` map
3. Add definition to `fieldDefinitions` array
4. Restart plugin (fields synced on activation)

#### 3.4.2 One-Time Field Sync (On Activation)

**Decision:** Fields are synced once during plugin activation, not on every periodic sync.

**Rationale:**
- Hardcoded fields are static (never change at runtime)
- Unnecessary API calls on every sync
- Reduces log noise
- Faster periodic syncs

**Implications:**
- Field changes require plugin restart
- Fields must exist before value sync runs
- OnActivate must complete successfully

**Error Handling:**
- Field sync failure prevents plugin activation
- Returns error to prevent background job from starting
- Clear error logs guide troubleshooting

#### 3.4.3 Field Type Immutability

**Decision:** Field types cannot be changed after creation.

**Constraint:** This is a Mattermost CPA platform limitation.

**Implications:**
- Changing field type requires deleting and recreating field
- All existing user values lost when field deleted
- Test schema carefully before production deployment

**Best Practice:** Use clear, descriptive field IDs and types from the start.

#### 3.4.4 Email-Based User Identity

**Decision:** Match external users to Mattermost users by email address.

**Rationale:**
- Most reliable cross-system identifier
- Available in most external systems
- No need for custom user mapping

**Limitations:**
- Email changes break synchronization
- Case-sensitive matching
- Users must have email in Mattermost

**Alternative:** For systems without email, modify `SyncUsers()` to use username, employee ID, or custom mapping.

#### 3.4.5 Minimal State Storage

**Decision:** Only store sync timestamp in KVStore (no field mappings, no option caches).

**Rationale:**
- Hardcoded mappings eliminate need for runtime lookups
- Simpler KVStore interface (2 methods vs 6)
- Less storage overhead
- Fewer potential consistency issues

**State Stored:**
- File modification time (for incremental sync)

**State NOT Stored:**
- Field name → Field ID mappings (hardcoded)
- Option name → Option ID mappings (hardcoded)
- Field definitions (in code)

#### 3.4.6 Graceful Degradation

**Decision:** Individual failures don't stop entire sync operation.

**Field Sync:**
- Failed field creation → log error, continue with next field
- Partial success is acceptable

**Value Sync:**
- User not found → log warning, skip user
- Unknown field → log warning, skip field
- Unknown option → log error, skip field
- Upsert failure → log error, skip user

**Rationale:**
- Maximizes progress despite data quality issues
- Prevents one bad record from blocking thousands
- Easier to diagnose issues from logs

---

## 4. AttributeProvider Interface

The `AttributeProvider` interface abstracts the external data source, making it easy to swap implementations (file, API, LDAP, database, etc.).

### 4.1 Interface Definition

```go
type AttributeProvider interface {
    // GetUserAttributes retrieves user attribute data from external source.
    // Returns array of user objects where each object is a map of field names to values.
    // Should track state internally to support incremental synchronization.
    // Returns empty array if no new/changed data available.
    GetUserAttributes() ([]map[string]interface{}, error)

    // Close releases resources held by provider (connections, file handles).
    Close() error
}
```

### 4.2 Data Format

**Required Fields:**
- `email` (string) - Used for Mattermost user resolution

**Attribute Fields:**
- `job_title` (string) - Text field
- `programs` ([]string or []interface{}) - Multiselect field
- `start_date` (string, YYYY-MM-DD) - Date field

**Example:**
```json
[
  {
    "email": "john.doe@example.com",
    "job_title": "Software Engineer",
    "programs": ["Apples", "Oranges"],
    "start_date": "2023-01-15"
  },
  {
    "email": "jane.smith@example.com",
    "job_title": "Sales Manager",
    "programs": ["Lemons"],
    "start_date": "2022-08-01"
  }
]
```

### 4.3 FileProvider Implementation

**Location:** `server/sync/file_provider.go`

**Data Source:** `data/user_attributes.json`

**Incremental Sync:**
- Tracks file modification time (`os.Stat()`)
- First call: returns all users
- Subsequent calls: returns users only if file modified
- If unchanged: returns empty array

**Example Usage:**
```go
provider := sync.NewFileProvider()
users, err := provider.GetUserAttributes()
// First call: returns all users
users, err = provider.GetUserAttributes()
// Second call: returns [] (file unchanged)
// ... modify file ...
users, err = provider.GetUserAttributes()
// Third call: returns all users (file modified)
```

### 4.4 Creating Custom Providers

To integrate with a different data source:

1. **Implement AttributeProvider interface**
2. **Handle incremental sync** (track last sync time, use change detection)
3. **Return correct data format** (map with email + attribute fields)
4. **Handle errors gracefully** (log and return error)

**Example API Provider:**
```go
type APIProvider struct {
    baseURL      string
    apiKey       string
    lastSyncTime time.Time
}

func (p *APIProvider) GetUserAttributes() ([]map[string]interface{}, error) {
    // Build request with lastSyncTime filter
    // Call external API
    // Parse response into []map[string]interface{}
    // Update lastSyncTime
    return users, nil
}

func (p *APIProvider) Close() error {
    return nil
}
```

---

## 5. Implementation Details

### 5.1 Hardcoded Field Definitions

**Location:** `server/sync/field_sync.go`

**Constants:**
```go
const (
    FieldIDJobTitle  = "field_job_title"
    FieldIDPrograms  = "field_programs"
    FieldIDStartDate = "field_start_date"

    OptionIDApples  = "option_apples"
    OptionIDOranges = "option_oranges"
    OptionIDLemons  = "option_lemons"
)
```

**Mappings:**
```go
// External field name → Mattermost field ID
var fieldNameToID = map[string]string{
    "job_title":  FieldIDJobTitle,
    "programs":   FieldIDPrograms,
    "start_date": FieldIDStartDate,
}

// Option name → Option ID (for programs field)
var programOptionNameToID = map[string]string{
    "Apples":  OptionIDApples,
    "Oranges": OptionIDOranges,
    "Lemons":  OptionIDLemons,
}
```

**Field Definitions:**
```go
var fieldDefinitions = []fieldDefinition{
    {
        ID:   FieldIDJobTitle,
        Name: "Job Title",
        Type: model.PropertyFieldTypeText,
    },
    {
        ID:   FieldIDPrograms,
        Name: "Programs",
        Type: model.PropertyFieldTypeMultiselect,
        Options: []map[string]interface{}{
            {"id": OptionIDApples, "name": "Apples"},
            {"id": OptionIDOranges, "name": "Oranges"},
            {"id": OptionIDLemons, "name": "Lemons"},
        },
    },
    {
        ID:   FieldIDStartDate,
        Name: "Start Date",
        Type: model.PropertyFieldTypeDate,
    },
}
```

### 5.2 Field Synchronization

**When:** Once during plugin activation (`OnActivate`)

**Function:** `SyncFields(client, groupID)`

**Behavior:**
1. Iterate through `fieldDefinitions` array
2. For each field, call `createOrUpdateField()`
3. Attempt to create field
4. If creation fails (already exists):
   - Get existing field
   - Update to match hardcoded definition
5. Continue with next field on error (graceful degradation)

**Idempotency:**
- Safe to run multiple times
- Updates existing fields to match definitions
- No duplicate fields created

### 5.3 Value Synchronization

**When:** Periodically every 60 minutes

**Function:** `SyncUsers(client, groupID, users)`

**Behavior:**
1. For each user in external data:
   - Extract email
   - Resolve Mattermost user by email
   - Build PropertyValues for all attributes
   - Bulk upsert values
2. Continue on individual user failures

**Value Building:**
- Call `buildPropertyValues()` for each user
- For each attribute:
  - Look up field ID from `fieldNameToID`
  - Format value based on type:
    - `[]string` → `formatMultiselectValue()` → option IDs
    - `string` → `formatStringValue()` → JSON-encoded string
  - Create PropertyValue object
- Return array of PropertyValues

**Multiselect Handling:**
- Convert option names to option IDs using `programOptionNameToID`
- Marshal option IDs to JSON array
- Unknown options cause error (schema mismatch)

### 5.4 Plugin Lifecycle

**OnActivate:**
```go
func (p *Plugin) OnActivate() error {
    // Initialize clients
    p.client = pluginapi.NewClient(p.API, p.Driver)
    p.kvstore = kvstore.NewKVStore(p.client)

    // Sync fields once
    groupID, err := sync.GetOrRegisterCPAGroup(p.client)
    if err != nil {
        return errors.Wrap(err, "failed to get CPA group")
    }

    err = sync.SyncFields(p.client, groupID)
    if err != nil {
        return errors.Wrap(err, "failed to sync fields")
    }

    // Start background job
    job, err := cluster.Schedule(p.API, "AttributeSync",
        p.nextWaitInterval, p.runSync)
    if err != nil {
        return errors.Wrap(err, "failed to schedule sync job")
    }

    p.backgroundJob = job
    return nil
}
```

**OnDeactivate:**
```go
func (p *Plugin) OnDeactivate() error {
    if p.backgroundJob != nil {
        p.backgroundJob.Close()
    }
    return nil
}
```

---

## 6. Customization Guide

### 6.1 Adding New Fields

**Steps:**
1. Add field ID constant
2. Add field name mapping
3. Add field definition
4. If multiselect, add option IDs and mappings
5. Restart plugin

**Example: Adding "team" Field**

```go
// 1. Add constant
const (
    FieldIDJobTitle  = "field_job_title"
    FieldIDPrograms  = "field_programs"
    FieldIDStartDate = "field_start_date"
    FieldIDTeam      = "field_team"  // NEW
)

// 2. Add mapping
var fieldNameToID = map[string]string{
    "job_title":  FieldIDJobTitle,
    "programs":   FieldIDPrograms,
    "start_date": FieldIDStartDate,
    "team":       FieldIDTeam,  // NEW
}

// 3. Add definition
var fieldDefinitions = []fieldDefinition{
    // ... existing fields ...
    {
        ID:   FieldIDTeam,
        Name: "Team",
        Type: model.PropertyFieldTypeText,
    },
}
```

### 6.2 Changing Field Types

**WARNING:** Cannot change field type after creation (Mattermost limitation).

**To Change Type:**
1. Use System Console to delete the old field (loses all user values)
2. Update field definition in code
3. Restart plugin (creates field with new type)

**Best Practice:** Test schema thoroughly before production deployment.

### 6.3 Adding Multiselect Options

**Steps:**
1. Add option ID constant
2. Add option name → ID mapping
3. Add option to field definition
4. Restart plugin

**Example: Adding "Bananas" Option**

```go
// 1. Add constant
const (
    OptionIDApples  = "option_apples"
    OptionIDOranges = "option_oranges"
    OptionIDLemons  = "option_lemons"
    OptionIDBananas = "option_bananas"  // NEW
)

// 2. Add mapping
var programOptionNameToID = map[string]string{
    "Apples":  OptionIDApples,
    "Oranges": OptionIDOranges,
    "Lemons":  OptionIDLemons,
    "Bananas": OptionIDBananas,  // NEW
}

// 3. Add to field definition
Options: []map[string]interface{}{
    {"id": OptionIDApples, "name": "Apples"},
    {"id": OptionIDOranges, "name": "Oranges"},
    {"id": OptionIDLemons, "name": "Lemons"},
    {"id": OptionIDBananas, "name": "Bananas"},  // NEW
},
```

### 6.4 Changing Sync Interval

**Location:** `server/job.go`

```go
// Change from 60 to desired minutes
const syncIntervalMinutes = 60
```

### 6.5 Implementing Custom Provider

**Example: LDAP Provider**

```go
type LDAPProvider struct {
    conn         *ldap.Conn
    baseDN       string
    filter       string
    lastSyncTime time.Time
}

func (p *LDAPProvider) GetUserAttributes() ([]map[string]interface{}, error) {
    // Build LDAP search request with modifyTimestamp filter
    searchRequest := ldap.NewSearchRequest(
        p.baseDN,
        ldap.ScopeWholeSubtree,
        ldap.NeverDerefAliases,
        0, 0, false,
        fmt.Sprintf("(&%s(modifyTimestamp>=%s))", p.filter, p.lastSyncTime),
        []string{"mail", "title", "memberOf", "createTimestamp"},
        nil,
    )

    // Execute search
    result, err := p.conn.Search(searchRequest)
    if err != nil {
        return nil, err
    }

    // Convert LDAP entries to user attribute maps
    users := make([]map[string]interface{}, 0, len(result.Entries))
    for _, entry := range result.Entries {
        user := map[string]interface{}{
            "email":      entry.GetAttributeValue("mail"),
            "job_title":  entry.GetAttributeValue("title"),
            "start_date": entry.GetAttributeValue("createTimestamp"),
            "programs":   entry.GetAttributeValues("memberOf"),
        }
        users = append(users, user)
    }

    p.lastSyncTime = time.Now()
    return users, nil
}

func (p *LDAPProvider) Close() error {
    if p.conn != nil {
        p.conn.Close()
    }
    return nil
}
```

**Update job.go:**
```go
func (p *Plugin) runSync() {
    // Replace FileProvider with LDAPProvider
    provider := sync.NewLDAPProvider(ldapURL, baseDN, filter)
    defer provider.Close()

    users, err := provider.GetUserAttributes()
    // ... rest of sync logic unchanged ...
}
```

---

## 7. Appendices

### Appendix A: Example Data Formats

**Input: user_attributes.json**
```json
[
  {
    "email": "john.doe@example.com",
    "job_title": "Software Engineer",
    "programs": ["Apples", "Oranges"],
    "start_date": "2023-01-15"
  },
  {
    "email": "jane.smith@example.com",
    "job_title": "Sales Manager",
    "programs": ["Lemons"],
    "start_date": "2022-08-01"
  }
]
```

**Output: PropertyValues (after formatting)**
```json
[
  {
    "group_id": "custom_profile_attributes",
    "target_type": "user",
    "target_id": "user123",
    "field_id": "field_job_title",
    "value": "\"Software Engineer\""
  },
  {
    "group_id": "custom_profile_attributes",
    "target_type": "user",
    "target_id": "user123",
    "field_id": "field_programs",
    "value": "[\"option_apples\",\"option_oranges\"]"
  },
  {
    "group_id": "custom_profile_attributes",
    "target_type": "user",
    "target_id": "user123",
    "field_id": "field_start_date",
    "value": "\"2023-01-15\""
  }
]
```

### Appendix B: Mattermost Property APIs

**GetPropertyGroup:**
```go
group, err := client.Property.GetPropertyGroup("custom_profile_attributes")
```

**CreatePropertyField:**
```go
field := &model.PropertyField{
    ID:      "field_job_title",
    GroupID: groupID,
    Name:    "Job Title",
    Type:    model.PropertyFieldTypeText,
    Attrs: model.StringInterface{
        model.CustomProfileAttributesPropertyAttrsVisibility:
            model.CustomProfileAttributesVisibilityHidden,
        model.CustomProfileAttributesPropertyAttrsManaged: "admin",
    },
}
createdField, err := client.Property.CreatePropertyField(field)
```

**UpdatePropertyField:**
```go
updatedField, err := client.Property.UpdatePropertyField(groupID, field)
```

**UpsertPropertyValues:**
```go
values := []*model.PropertyValue{
    {
        GroupID:    groupID,
        TargetType: "user",
        TargetID:   userID,
        FieldID:    "field_job_title",
        Value:      json.RawMessage("\"Software Engineer\""),
    },
}
upsertedValues, err := client.Property.UpsertPropertyValues(values)
```

### Appendix C: Error Handling Examples

**Field Creation Failure (Idempotent Retry):**
```go
_, err := client.Property.CreatePropertyField(field)
if err != nil {
    // Field might already exist - try to get and update
    existingField, getErr := client.Property.GetPropertyField(groupID, fieldID)
    if getErr != nil {
        return errors.Wrap(err, "field doesn't exist and creation failed")
    }
    // Update existing field to match definition
    _, updateErr := client.Property.UpdatePropertyField(groupID, existingField)
    if updateErr != nil {
        return errors.Wrap(updateErr, "failed to update existing field")
    }
}
```

**User Not Found (Graceful Skip):**
```go
user, err := api.User.GetByEmail(email)
if err != nil {
    client.Log.Warn("User not found by email, skipping",
        "email", email,
        "error", err.Error())
    continue // Skip this user, process next
}
```

**Unknown Field (Warning Log):**
```go
fieldID := GetFieldID(fieldName)
if fieldID == "" {
    client.Log.Warn("Unknown field name, skipping",
        "field_name", fieldName,
        "user_email", user.Email)
    continue // Skip this field, process next
}
```

**Unknown Option (Error):**
```go
optionID := GetProgramOptionID(optionName)
if optionID == "" {
    return fmt.Errorf("unknown program option: %s", optionName)
}
```

---

## Document History

- **Initial Version:** Dynamic field discovery approach with type inference
- **Current Version:** Hardcoded fields approach for simplicity and clarity
- **Last Updated:** 2025-11-21

**Key Changes:**
- Removed dynamic field discovery and type inference
- Simplified to hardcoded field schema
- One-time field sync on plugin activation
- Minimal KVStore usage (timestamp only)
- ~80% code reduction for better maintainability

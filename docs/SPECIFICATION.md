# User Attribute Sync Starter Template - Specification

## 1. Overview

### 1.1 Purpose and Scope

This specification defines the **User Attribute Sync Starter Template**, a reference implementation Mattermost plugin that demonstrates how to synchronize user profile attributes from external systems into Mattermost's Custom Profile Attributes (CPA) system.

**Primary Purpose:**
Provide plugin developers with a working, well-documented example of how to:
- Fetch user attribute data from external systems
- Dynamically create Custom Profile Attribute fields based on data structure
- Synchronize attribute values for users in an incremental, efficient manner
- Handle multiselect options management and type inference
- Implement cluster-aware background jobs for reliable synchronization

**Scope:**
This is a **starter template** and **educational resource**, not a production-ready integration. It demonstrates best practices and patterns that developers should adapt for their specific use cases.

**In Scope:**
- Dynamic field creation from JSON data structure
- Type inference (text, date, multiselect)
- Incremental user attribute synchronization
- Append-only multiselect option management
- Cluster-aware background job scheduling
- Abstracted data source interface (swap file/API/LDAP)
- Comprehensive error handling and logging patterns
- Email-based user identification

**Out of Scope:**
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
1. **Dynamic Schema Discovery**
   - Automatically create PropertyFields from JSON structure
   - Infer field types from data (text, multiselect, date)
   - Transform field names to user-friendly display names

2. **Incremental Synchronization**
   - Process only changed users after initial sync
   - Track sync state in KVStore
   - Handle partial failures gracefully

3. **Multiselect Option Management**
   - Append-only option accumulation
   - Preserve option IDs across syncs
   - Detect and add new option values dynamically

4. **Data Source Abstraction**
   - Interface-based provider pattern
   - Easy to swap file/API/database implementations
   - Example file-based provider included

#### Mattermost Plugin Patterns
5. **Cluster-Aware Jobs**
   - Use Mattermost cluster job system
   - Prevent duplicate work in multi-server deployments
   - Configurable sync intervals

6. **Property API Usage**
   - Create and update PropertyFields
   - Bulk upsert PropertyValues efficiently
   - Query and manage field options

7. **Error Handling**
   - Graceful degradation for partial failures
   - Comprehensive structured logging
   - Retry-safe idempotent operations

8. **State Management**
   - Use KVStore for persistent state
   - Track field mappings and metadata
   - Store accumulated options

#### Code Quality
9. **Well-Documented Code**
   - Extensive inline comments
   - Clear separation of concerns
   - Reusable, testable components

10. **Testing Strategies**
    - Unit tests for core logic
    - Mock external data sources
    - Example test data included

### 1.4 Relationship to Broader Initiative

This template is part of the **External Attribute System Connector** initiative (MM-65781), which aims to provide a standardized way for enterprises to synchronize Custom Profile Attributes with external identity and attribute management systems.

**Future Enhancements (Not in This Template):**
- `ExternallyManaged` flag on PropertyFields (requires core Mattermost changes)
- System Console UI for externally-managed field indicators
- Conflict resolution for competing external sources
- Bi-directional sync capabilities

**Why a Starter Template?**
Rather than building a one-size-fits-all integration plugin, this template provides a foundation that developers can customize for their specific:
- External system APIs and data formats
- Authentication mechanisms
- Data transformation requirements
- Error handling policies
- Performance and scale needs

### 1.5 How to Use This Specification

**For Plugin Developers:**
1. Read sections 1-4 to understand the architecture and design
2. Review section 5 to understand the AttributeProvider interface
3. Study section 6 for implementation details
4. Use section 8 as a guide when customizing for your external system
5. Reference appendices for example data formats and API details

**For System Architects:**
- Focus on sections 2-4 for high-level design and requirements
- Review section 4.4 (Design Constraints) to understand tradeoffs
- Use this as a reference for planning custom integrations

**For Code Reviewers:**
- Section 6 provides detailed implementation specifications
- Section 7 defines expected operational behavior
- Section 9 outlines testing requirements

## 2. Problem Statement

### 2.1 Current Challenges

Organizations using Mattermost in enterprise environments face significant challenges when attempting to synchronize user attributes from external identity and attribute management systems (such as HR systems, LDAP directories, or User Access Services) with Mattermost's Custom Profile Attributes.

#### Challenge 1: No Standardized Integration Path

Currently, there is no established pattern or reference implementation for synchronizing Custom Profile Attributes with external systems. Each integration requires:
- Custom plugin development from scratch
- Discovery of relevant Mattermost PropertyService APIs through documentation
- Trial and error to understand PropertyField and PropertyValue relationships
- Reinventing solutions for common problems (option management, type mapping, incremental sync)

**Impact:**
- Slow time-to-integration for enterprises
- Inconsistent implementation patterns across different plugins
- Higher maintenance burden due to lack of best practices

#### Challenge 2: Manual Synchronization Overhead

Without automated synchronization, administrators must:
- Manually create Custom Profile Attribute field definitions in System Console
- Manually update field options when new values appear in external systems
- Manually set user attribute values or rely on users to self-report
- Continuously monitor for data drift between systems

**Impact:**
- Data inconsistencies between source-of-truth systems and Mattermost
- Administrative burden scales with organization size
- Stale attribute data reduces effectiveness of Attribute-Based Access Control (ABAC)
- Risk of human error in data entry

#### Challenge 3: Complex Schema Management

Different organizations have vastly different attribute schemas:
- HR systems may have dozens of custom fields
- LDAP directories use different attribute naming conventions
- Some attributes are simple text, others are multi-valued enumerations
- Schema evolution (new fields, new values) is common

Developers need to understand:
- How to map external types to Mattermost PropertyField types
- How to handle multiselect options and preserve IDs across updates
- How to detect and handle schema changes
- When to create vs update fields

**Impact:**
- High learning curve for plugin developers
- Brittle integrations that break with schema changes
- Orphaned data when field definitions change

#### Challenge 4: Scale and Performance Concerns

Enterprise deployments may have:
- Tens of thousands of users
- Dozens of attributes per user
- Frequent attribute updates (daily or hourly)
- Multi-server clusters requiring coordination

Without proper patterns:
- Naive implementations sync all users on every run (inefficient)
- Duplicate work in cluster environments
- Database strain from excessive API calls
- Long sync times blocking other operations

**Impact:**
- Poor performance and resource utilization
- Potential for race conditions and data corruption
- Inability to sync at desired frequency

### 2.2 Why a Starter Template Is Needed

#### Accelerate Integration Development

A well-documented starter template provides:
- **Working reference implementation** that developers can run and study
- **Proven patterns** for common tasks (type inference, option management, incremental sync)
- **Best practices** for Mattermost plugin development
- **Reusable components** that can be adapted to specific external systems

Instead of spending weeks learning Mattermost APIs and debugging integration issues, developers can focus on:
- Adapting the data source layer to their external system
- Customizing data transformation logic
- Adding authentication for their specific API

#### Reduce Risk and Improve Quality

The template demonstrates:
- **Proper error handling** (partial failures, retries, graceful degradation)
- **Cluster-aware operations** (prevents duplicate work, handles failover)
- **Idempotent sync operations** (safe to retry, no data corruption)
- **Comprehensive logging** (observability and debugging)

This reduces risk of:
- Data loss or corruption
- Performance problems in production
- Difficult-to-debug integration issues

#### Enable Broader Adoption of ABAC

Custom Profile Attributes are a key component of Mattermost's Attribute-Based Access Control capabilities. However, ABAC is most valuable when attributes are:
- Automatically synchronized from authoritative sources
- Kept up-to-date with organizational changes
- Consistently defined across the organization

By lowering the barrier to attribute synchronization, this template enables more organizations to:
- Leverage their existing identity infrastructure
- Implement sophisticated access control policies
- Maintain data consistency across systems

### 2.3 Relationship to MM-65781 Initiative

This starter template is part of the broader **External Attribute System Connector** initiative (MM-65781), which has a vision to:

> Provide a standardized plugin framework that enables synchronization between external attribute providers and our Custom Profile Attribute system, with clear ownership boundaries and external management indicators.

**This Template's Role:**

The starter template addresses the initiative's goals by:

1. **Providing a standardized integration path** - Establishes patterns and interfaces that future integrations can follow
2. **Demonstrating best practices** - Shows how to use PropertyService APIs correctly and efficiently
3. **Enabling rapid development** - Reduces integration time from weeks to days
4. **Supporting the ABAC value proposition** - Makes it easier for customers to populate attributes from authoritative sources

**Complementary Components (Out of Scope for Template):**

The broader initiative includes additional components that require Mattermost core changes:

- **ExternallyManaged flag on PropertyFields** - Marks fields as read-only in UI when managed by plugins
- **System Console UI enhancements** - Visual indicators showing which fields are externally managed
- **Plugin API extensions** - Additional methods for querying externally-managed fields
- **Conflict resolution** - Handling multiple plugins attempting to manage the same fields

These components are **not included** in this starter template, which focuses purely on the synchronization mechanics that work with current Mattermost capabilities.

### 2.4 Target Use Cases

This template is designed to support integration with:

**Identity Management Systems:**
- LDAP/Active Directory (user groups, departments, locations)
- SAML Identity Providers (roles, clearances, affiliations)
- HR Systems (employee status, job title, manager)
- Access Management Systems (permissions, certifications, training)

**Common Integration Scenarios:**
- Sync user department and location for geographic access policies
- Sync security clearances for classified channel access
- Sync project memberships for team-based access control
- Sync manager relationships for approval workflows
- Sync training certifications for compliance-based access

**Example: HR System Integration**
```
External HR System Attributes:
- employee_id
- department (Engineering, Sales, Marketing)
- location (US-East, US-West, EMEA, APAC)
- employment_type (Full-time, Contractor, Intern)
- security_clearance (Unclassified, Confidential, Secret)

Sync Flow:
1. Plugin fetches changed users from HR API
2. Dynamically creates CPA fields for each attribute
3. Syncs values for all changed users
4. Admins use these attributes in channel access policies
```

### 2.5 Success Criteria

This template will be considered successful if it:

**For Developers:**
- Reduces integration development time by 70% compared to starting from scratch
- Provides clear, working examples of all key synchronization patterns
- Can be adapted to different external systems with < 1 day of effort

**For Organizations:**
- Enables reliable, automated attribute synchronization at enterprise scale
- Maintains data consistency between external systems and Mattermost
- Supports ABAC use cases without manual attribute management

**For the Mattermost Ecosystem:**
- Establishes a consistent pattern for attribute sync plugins
- Reduces support burden by codifying best practices
- Accelerates adoption of Custom Profile Attributes and ABAC features

## 3. Goals and Requirements

### 3.1 Primary Objectives

#### Objective 1: Provide a Working Reference Implementation
Deliver a fully functional plugin that developers can:
- Install and run immediately with example data
- Study to understand attribute synchronization patterns
- Use as a foundation for their own integrations
- Reference when debugging their implementations

**Success Metrics:**
- Plugin builds and runs without errors on supported Mattermost versions
- Example data successfully syncs to Custom Profile Attributes
- All core synchronization patterns are demonstrated in working code

#### Objective 2: Demonstrate Best Practices
Showcase proper usage of:
- Mattermost Plugin API (PropertyService, cluster jobs, KVStore)
- Error handling and recovery patterns
- Logging and observability practices
- Code organization and separation of concerns
- Testing strategies for plugin components

**Success Metrics:**
- Code follows Mattermost plugin development guidelines
- Comprehensive inline documentation explains design decisions
- Error scenarios are handled gracefully with clear logging
- Components are testable with example unit tests

#### Objective 3: Enable Rapid Customization
Provide clear abstraction points where developers can:
- Swap the data source (file → REST API → LDAP → database)
- Customize type inference logic
- Add authentication mechanisms
- Modify data transformation rules
- Extend field types or validation

**Success Metrics:**
- AttributeProvider interface is well-defined and documented
- Example showing how to implement a custom provider
- < 1 day of effort to adapt template to new external system
- No need to modify core sync logic when changing data source

#### Objective 4: Support Enterprise Scale
Demonstrate patterns that work with:
- Thousands of users
- Dozens of attributes per user
- Multi-server cluster deployments
- Frequent synchronization intervals (hourly or more)

**Success Metrics:**
- Uses incremental sync (only changed users after first run)
- Cluster-aware job prevents duplicate work
- Bulk API operations minimize database load
- Idempotent operations support safe retries

### 3.2 Functional Requirements

#### FR1: Dynamic Field Creation
**Requirement:** The plugin SHALL automatically create Custom Profile Attribute fields based on the structure of user data from the external source.

**Details:**
- Extract unique field names from user objects (excluding "email")
- Infer field type from data (text, multiselect, date)
- Create PropertyFields with appropriate types
- Transform field names to user-friendly display names (title case)
- Store field name → ID mappings in KVStore

**Acceptance Criteria:**
- Given user data with new field "department", system creates text PropertyField named "department" with display name "Department"
- Given user data with array field "security_clearance", system creates multiselect PropertyField
- Field mappings persist across plugin restarts

#### FR2: Type Inference
**Requirement:** The plugin SHALL infer PropertyField types from JSON data structure.

**Details:**
- JSON array → `PropertyFieldTypeMultiselect`
- JSON string matching date pattern (YYYY-MM-DD) → `PropertyFieldTypeDate`
- JSON string (other) → `PropertyFieldTypeText`
- Type inference occurs only at field creation (no type changes)

**Acceptance Criteria:**
- String "2023-01-15" inferred as date field
- Array ["Level1", "Level2"] inferred as multiselect field
- String "Engineering" inferred as text field
- Once created, field type never changes

#### FR3: Incremental User Synchronization
**Requirement:** The plugin SHALL support incremental synchronization, processing only users whose attributes have changed since the last successful sync.

**Details:**
- Store last sync timestamp in KVStore
- Pass timestamp to AttributeProvider on subsequent syncs
- Provider returns only users with changes since that timestamp
- First sync processes all users (no timestamp)

**Acceptance Criteria:**
- First sync processes all 1000 users
- Second sync processes only 15 users who changed
- Last sync timestamp updates only on successful sync completion

#### FR4: Append-Only Multiselect Options
**Requirement:** The plugin SHALL accumulate multiselect options over time, never removing options from field definitions.

**Details:**
- Query existing field to get current options
- Collect new option values from user data
- Add new options while preserving existing option IDs
- Update field definition with expanded option set
- Store accumulated options in KVStore

**Acceptance Criteria:**
- Sync 1 creates field with options ["Eng", "Sales"]
- Sync 2 encounters new value "Marketing"
- Field updated to ["Eng", "Sales", "Marketing"] with original IDs preserved
- Option "Eng" retains same ID across both syncs

#### FR5: Email-Based User Resolution
**Requirement:** The plugin SHALL identify Mattermost users by email address.

**Details:**
- Each user object MUST contain an "email" field
- Use `API.GetUserByEmail()` to resolve to Mattermost user ID
- Skip users not found in Mattermost (with warning log)
- Continue processing remaining users after skip

**Acceptance Criteria:**
- User object with email "john@example.com" syncs to corresponding Mattermost user
- User object with email not in Mattermost logs warning and is skipped
- Skipped users don't cause sync to fail

#### FR6: Cluster-Aware Scheduling
**Requirement:** The plugin SHALL use Mattermost cluster jobs to ensure only one server instance performs synchronization in multi-server deployments.

**Details:**
- Use `cluster.Schedule()` for job management
- Configurable sync interval (default: 1 hour)
- Job restarts automatically on plugin activation
- Job stops cleanly on plugin deactivation

**Acceptance Criteria:**
- In 3-server cluster, only 1 server runs sync job
- If active server fails, another server takes over within 1 minute
- Job interval adjustable via plugin configuration
- No duplicate syncs occur

#### FR7: Partial Failure Handling
**Requirement:** The plugin SHALL continue processing remaining users when individual user sync operations fail.

**Details:**
- User email not found → log warning, skip user, continue
- Field lookup fails → log error, skip field for that user, continue
- Value upsert fails → log error, continue with next user
- Track and report success/failure counts

**Acceptance Criteria:**
- Sync with 100 users where 3 emails not found completes successfully
- Sync summary reports: "97/100 users synced (3 skipped)"
- Failed users don't prevent subsequent users from syncing

#### FR8: State Persistence
**Requirement:** The plugin SHALL persist synchronization state in KVStore to survive restarts.

**Details:**
- Store field name → Mattermost field ID mappings
- Store accumulated multiselect options per field
- Store last successful sync timestamp
- Store sync statistics (optional, for observability)

**Acceptance Criteria:**
- Plugin restart preserves field mappings (no duplicate fields created)
- Plugin restart preserves accumulated options
- Incremental sync after restart uses correct timestamp

### 3.3 Non-Functional Requirements

#### NFR1: Performance
**Requirement:** The plugin SHALL sync 1000 users with 10 attributes each in under 30 seconds.

**Details:**
- Use bulk `UpsertPropertyValues()` operations
- Minimize individual API calls
- Batch operations per user

#### NFR2: Observability
**Requirement:** The plugin SHALL provide comprehensive structured logging at all stages of synchronization.

**Details:**
- Log levels: DEBUG (detailed operations), INFO (summary), WARN (recoverable issues), ERROR (failures)
- Include context: user emails, field names, counts, durations
- Log sync start, completion, and summary statistics

#### NFR3: Code Quality
**Requirement:** The plugin code SHALL be well-documented with inline comments explaining design decisions and patterns.

**Details:**
- Comments explain "why" not just "what"
- Complex logic includes examples
- Public interfaces have godoc comments
- README provides architecture overview

#### NFR4: Testability
**Requirement:** The plugin SHALL include unit tests demonstrating how to test core synchronization logic.

**Details:**
- Mock AttributeProvider for testing
- Example tests for type inference
- Example tests for option management
- Test data included in repository

### 3.4 In Scope

**Core Synchronization:**
- ✅ Dynamic field creation from JSON structure
- ✅ Type inference (text, date, multiselect)
- ✅ Incremental user synchronization
- ✅ Append-only multiselect option management
- ✅ Email-based user resolution
- ✅ Bulk PropertyValue upserts

**Infrastructure:**
- ✅ Cluster-aware background job scheduling
- ✅ KVStore-based state management
- ✅ Configurable sync intervals
- ✅ Comprehensive logging

**Developer Experience:**
- ✅ AttributeProvider interface abstraction
- ✅ File-based reference implementation
- ✅ Example JSON data files
- ✅ Extensive inline documentation
- ✅ Unit test examples

**Documentation:**
- ✅ Architecture specification (this document)
- ✅ README with setup instructions
- ✅ Integration guide for developers
- ✅ Example data formats

### 3.5 Out of Scope

**Advanced Features:**
- ❌ ExternallyManaged flag on PropertyFields (requires core Mattermost changes)
- ❌ System Console UI for monitoring sync status
- ❌ Webhook endpoint for on-demand sync triggers
- ❌ Slash commands for manual sync control
- ❌ Bi-directional synchronization (Mattermost → external system)
- ❌ User provisioning/deprovisioning
- ❌ Field deletion or type changes
- ❌ Conflict resolution for multiple sources managing same fields

**Production Features:**
- ❌ Advanced error recovery (exponential backoff, circuit breakers)
- ❌ Metrics and monitoring integrations (Prometheus, etc.)
- ❌ Health check endpoints
- ❌ Rate limiting for external API calls
- ❌ Retry queues for failed operations
- ❌ Data validation beyond type checking

**External System Specifics:**
- ❌ Authentication implementations (OAuth, LDAP bind, etc.)
- ❌ Specific external API clients (HR systems, LDAP, etc.)
- ❌ Data transformation for specific systems
- ❌ External system-specific error handling

**Rationale:** These features are either:
1. System-specific (developers add when adapting template)
2. Require Mattermost core changes (future work)
3. Beyond the scope of a starter template (production hardening)

### 3.6 Constraints and Assumptions

#### Constraints

**Technical Constraints:**
- Must work with Mattermost server version 10.10+ (PropertyService APIs introduced)
- Must use only public Plugin API (no direct database access)
- Server-only plugin (no webapp component)
- Limited to Custom Profile Attributes property group

**Operational Constraints:**
- Sync frequency limited by external system capabilities
- User count limited by practical sync duration requirements
- Field count limited by Custom Profile Attributes (20 fields max)

#### Assumptions

**About External Systems:**
- External system can provide list of changed users since timestamp
- User email addresses are unique and stable
- Attribute data is provided in JSON format (or easily converted)
- External system is available during sync window

**About Mattermost Environment:**
- Users already exist in Mattermost before attribute sync
- User emails in Mattermost match external system emails
- Custom Profile Attributes feature is enabled
- Plugin has permissions to create/update PropertyFields and PropertyValues

**About Integrators:**
- Have Go programming knowledge
- Understand their external system's API
- Can adapt AttributeProvider interface to their system
- Will test thoroughly before production deployment

### 3.7 Dependencies

**Mattermost Server:**
- Version 10.10 or higher
- PropertyService API availability
- Custom Profile Attributes feature enabled

**Go Modules:**
- `github.com/mattermost/mattermost/server/public/plugin` - Plugin API
- `github.com/mattermost/mattermost/server/public/pluginapi` - Helper APIs
- `github.com/mattermost/mattermost/server/public/pluginapi/cluster` - Cluster jobs
- `github.com/mattermost/mattermost/server/public/model` - Data models

**No External Dependencies:**
- Template uses only Mattermost APIs and Go standard library
- Integrators add external dependencies as needed for their systems

### 3.8 Success Criteria

This implementation will be considered successful when:

**Functional Success:**
- [ ] Plugin installs and activates without errors
- [ ] Example data successfully syncs to Custom Profile Attributes
- [ ] Dynamic fields created with correct types
- [ ] Incremental sync processes only changed users
- [ ] Multiselect options accumulate correctly across syncs
- [ ] Cluster job runs on only one server instance

**Quality Success:**
- [ ] Code passes all unit tests
- [ ] Linting and formatting checks pass
- [ ] No critical or high-severity security issues
- [ ] Documentation is complete and accurate

**Developer Experience Success:**
- [ ] Developer can swap data source in < 4 hours
- [ ] Clear examples for all major patterns
- [ ] Inline comments explain non-obvious decisions
- [ ] README provides quick-start guide

**Community Validation:**
- [ ] Reviewed by 2+ Mattermost plugin developers
- [ ] Successfully adapted by at least one external integration
- [ ] Positive feedback on documentation clarity

## 4. Architecture

### 4.1 High-Level Design

The User Attribute Sync plugin follows a layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                   Mattermost Server                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │           Plugin Framework (Hooks)                      │ │
│  └────────────────────────────────────────────────────────┘ │
│                            ▲                                 │
│                            │                                 │
│  ┌────────────────────────┴───────────────────────────────┐ │
│  │        User Attribute Sync Plugin                      │ │
│  │                                                         │ │
│  │  ┌────────────────────────────────────────────────┐   │ │
│  │  │  Plugin Lifecycle (plugin.go)                   │   │ │
│  │  │  - OnActivate / OnDeactivate                    │   │ │
│  │  │  - Configuration Management                     │   │ │
│  │  └────────────────────────────────────────────────┘   │ │
│  │                            │                           │ │
│  │  ┌────────────────────────┴────────────────────────┐  │ │
│  │  │  Cluster Job (job.go)                           │  │ │
│  │  │  - Cluster-aware scheduling                     │  │ │
│  │  │  - Orchestrates sync workflow                   │  │ │
│  │  │  - Error handling & logging                     │  │ │
│  │  └─────────────────────────────────────────────────┘  │ │
│  │                            │                           │ │
│  │           ┌────────────────┴────────────────┐          │ │
│  │           ▼                                 ▼          │ │
│  │  ┌──────────────────┐           ┌─────────────────┐  │ │
│  │  │  Field Sync      │           │  Value Sync     │  │ │
│  │  │  (field_sync.go) │           │ (value_sync.go) │  │ │
│  │  │                  │           │                 │  │ │
│  │  │ - Infer types    │           │ - Email lookup  │  │ │
│  │  │ - Create fields  │           │ - Upsert values │  │ │
│  │  │ - Manage options │           │ - Type handling │  │ │
│  │  └────────┬─────────┘           └────────┬────────┘  │ │
│  │           │                              │            │ │
│  │           └────────────┬─────────────────┘            │ │
│  │                        ▼                              │ │
│  │           ┌───────────────────────┐                  │ │
│  │           │  AttributeProvider    │                  │ │
│  │           │  Interface            │                  │ │
│  │           │  GetUserAttributes()  │                  │ │
│  │           └───────────────────────┘                  │ │
│  │                        │                              │ │
│  │           ┌────────────┴────────────┐                 │ │
│  │           ▼                         ▼                 │ │
│  │  ┌──────────────┐         ┌──────────────────┐      │ │
│  │  │ File-Based   │         │ Future: REST API │      │ │
│  │  │ Provider     │         │ LDAP, Database   │      │ │
│  │  │ (default)    │         │ etc.             │      │ │
│  │  └──────────────┘         └──────────────────┘      │ │
│  │                                                       │ │
│  │  ┌─────────────────────────────────────────────┐    │ │
│  │  │  KVStore                                     │    │ │
│  │  │  - Field mappings (name → ID)               │    │ │
│  │  │  - Accumulated multiselect options          │    │ │
│  │  │  - Last sync timestamp                      │    │ │
│  │  └─────────────────────────────────────────────┘    │ │
│  └───────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
                 ┌────────────────────┐
                 │  Mattermost        │
                 │  PropertyService   │
                 │  API               │
                 └────────────────────┘
```

**Design Principles:**
- **Cluster-Aware**: Uses Mattermost cluster jobs to ensure only one instance syncs in multi-server deployments
- **Dynamic Schema**: Fields are created on-the-fly from JSON structure, no predefined schema required
- **Data Source Agnostic**: AttributeProvider interface enables swapping file/API/LDAP sources
- **Incremental Updates**: Only changed users are processed after initial sync
- **Append-Only Options**: Multiselect options accumulate over time, never removed
- **Separation of Concerns**: Field sync and value sync are independent operations
- **Observability**: Structured logging at all layers

### 4.2 Component Overview

#### Core Components

| Component | File | Responsibility |
|-----------|------|----------------|
| **Plugin** | `server/plugin.go` | Manages plugin lifecycle, configuration changes, and coordinates cluster job |
| **Configuration** | `server/configuration.go` | Handles plugin settings (sync interval, data source config) |
| **Cluster Job** | `server/job.go` | Orchestrates periodic sync using Mattermost cluster job system |
| **Field Synchronizer** | `server/sync/field_sync.go` | Infers field types, creates PropertyFields, manages multiselect options |
| **Value Synchronizer** | `server/sync/value_sync.go` | Resolves users by email, formats values, upserts PropertyValues |
| **AttributeProvider Interface** | `server/sync/provider.go` | Abstract interface for data sources |
| **File Provider** | `server/sync/file_provider.go` | Implements AttributeProvider using JSON files |
| **Type Inference** | `server/sync/types.go` | Detects field types from JSON values (text, multiselect, date) |
| **Data Models** | `server/sync/models.go` | Defines UserAttributes and related structures |

#### Supporting Components

| Component | File | Purpose |
|-----------|------|---------|
| **KVStore** | `server/store/kvstore/` | Stores field mappings, accumulated options, sync metadata |
| **Logger** | Via `pluginapi.Client.Log` | Structured logging throughout the plugin |

### 4.3 Data Flow

```
External Data Source (JSON/REST API/etc.)
         │
         │ 1. Fetch Changed Users
         ▼
┌────────────────────┐
│ AttributeProvider  │
│   Interface        │
│ GetUserAttributes()│
└────────────────────┘
         │
         │ 2. Returns array of user objects
         │    (only changed users after first sync)
         ▼
┌────────────────────┐
│   Cluster Job      │
│  (Orchestrator)    │
└────────────────────┘
         │
         ├─────────────────────┬────────────────────────┐
         │                     │                        │
         │ 3a. Sync Fields     │ 3b. Sync Values        │
         ▼                     ▼                        │
┌──────────────────┐  ┌──────────────────────┐        │
│  Field Sync      │  │   Value Sync         │        │
│                  │  │                      │        │
│ For each unique  │  │ For each user:       │        │
│ JSON key:        │  │ - Lookup by email    │        │
│ - Infer type     │  │ - Format values      │        │
│ - Create/update  │  │ - Build PropertyVals │        │
│ - Add options    │  │ - Bulk upsert        │        │
└──────────────────┘  └──────────────────────┘        │
         │                     │                        │
         │ 4a. PropertyField   │ 4b. PropertyValue      │
         │     CRUD            │     Upsert             │
         ▼                     ▼                        │
┌─────────────────────────────────────────────┐        │
│      Mattermost PropertyService API         │        │
│                                             │        │
│  - CreatePropertyField                      │        │
│  - UpdatePropertyField (for new options)    │        │
│  - UpsertPropertyValues                     │        │
└─────────────────────────────────────────────┘        │
         │                                              │
         │ 5. Store Metadata                            │
         ▼                                              │
┌────────────────────┐                                 │
│     KVStore        │                                 │
│ - Field name → ID  │◄────────────────────────────────┘
│ - Field options    │    6. Update state
│ - Last sync time   │
└────────────────────┘
```

### 4.4 Sync Workflow

#### Cluster Job Lifecycle

The plugin uses Mattermost's cluster job system for reliable, cluster-aware scheduling:

```go
// OnActivate in plugin.go
job, err := cluster.Schedule(
    p.API,
    "AttributeSync",
    p.nextWaitInterval,  // Calculates wait based on config interval
    p.runSync,           // Job execution function
)
```

**Benefits:**
- **Cluster-Aware**: Only one server instance runs the job, preventing duplicate work
- **Leader Election**: Automatic failover if the running instance fails
- **Configurable Interval**: Adjusts based on plugin configuration
- **Clean Lifecycle**: Proper start/stop on plugin activation/deactivation

#### Sync Execution Flow

##### Step 1: Fetch Changed Users

```
1. Cluster job scheduler invokes p.runSync()
2. Load plugin configuration
3. Initialize AttributeProvider
4. Call provider.GetUserAttributes()
   └─> Returns: Array of user objects with changed attributes
       - First sync: All users
       - Subsequent syncs: Only users changed since last sync timestamp

Example response:
[
  {
    "email": "john@example.com",
    "department": "Engineering",
    "security_clearance": ["Level2", "Level3"],
    "start_date": "2023-01-15"
  },
  {
    "email": "jane@example.com",
    "department": "Sales",
    "security_clearance": ["Level1"]
  }
]
```

##### Step 2: Field Discovery and Type Inference

```
1. Scan all user objects to discover unique field names
   └─> Extract all JSON keys except "email"
   └─> Result: ["department", "security_clearance", "start_date"]

2. For each field name, infer type from values:

   Type Inference Rules:
   - JSON array → PropertyFieldTypeMultiselect
   - JSON string matching date pattern → PropertyFieldTypeDate
   - JSON string (other) → PropertyFieldTypeText

   Examples:
   "department": "Engineering"           → text
   "security_clearance": ["Level2", ...] → multiselect
   "start_date": "2023-01-15"           → date

3. Transform field name to display name:
   "security_clearance" → "Security Clearance" (title case)
```

##### Step 3: Field Synchronization

```
For each discovered field:

  1. Check if field exists
     └─> Query KVStore for field name → Mattermost field ID mapping

  2a. If field does NOT exist:
      - Create new PropertyField
        * Name: original JSON key
        * Type: inferred type
        * Display name: title-cased transformation
      - For multiselect: initialize with empty options list
      - Store mapping in KVStore (name → ID)
      - Log: "Created field: {name} (type: {type})"

  2b. If field exists:
      - Skip (no type changes allowed)
      - Continue to options management

  3. For multiselect fields only - Manage Options:
     a. Query current field definition to get existing options
     b. Collect all unique values from user data for this field
     c. Compare with existing options:
        - Existing option with same name → reuse existing ID
        - New value → generate new option ID
     d. If new options found:
        - Build updated options list (existing + new)
        - Update PropertyField via UpdatePropertyField()
        - Store updated options in KVStore
        - Log: "Added {count} new options to {field_name}"

  4. Handle errors:
     - Log error details with context
     - Continue with next field (graceful degradation)

Result: All fields exist in Mattermost with current options
```

**Example - Multiselect Options Management:**
```
Sync 1:
  Users: [
    {"security_clearance": ["Level1", "Level2"]},
    {"security_clearance": ["Level2"]}
  ]
  Action: Create field with options: [
    {"id": "opt_abc123", "name": "Level1"},
    {"id": "opt_def456", "name": "Level2"}
  ]

Sync 2:
  Users: [
    {"security_clearance": ["Level3", "Level1"]}
  ]
  Action: Update field, add option: [
    {"id": "opt_abc123", "name": "Level1"},  // preserved
    {"id": "opt_def456", "name": "Level2"},  // preserved
    {"id": "opt_ghi789", "name": "Level3"}   // new
  ]
```

##### Step 4: Value Synchronization

```
For each user object in the response:

  1. Resolve user by email
     └─> Call API.GetUserByEmail(email)

  2. If user not found:
     - Log: WARN "User not found: {email}, skipping"
     - Continue to next user (don't fail sync)

  3. For each field in user object (except "email"):

     a. Look up PropertyField ID from KVStore mapping
        - If field not found: Log error, skip this field

     b. Format value based on field type:

        Text:
          value = "Engineering"
          formatted = json.Marshal("Engineering")

        Multiselect:
          value = ["Level2", "Level3"]
          1. Look up option IDs for each value name
          2. formatted = json.Marshal(["opt_def456", "opt_ghi789"])

        Date:
          value = "2023-01-15"
          formatted = json.Marshal("2023-01-15")  // ISO 8601

     c. Build PropertyValue object:
        {
          GroupID: "custom_profile_attributes",
          TargetType: "user",
          TargetID: user.Id,
          FieldID: field_id,
          Value: formatted_value
        }

  4. Batch all PropertyValues for this user

  5. Upsert values via PropertyService.UpsertPropertyValues()
     - Upsert = Create if new, Update if exists
     - Atomic operation per user

  6. Log result:
     - Success: "Synced {count} attributes for {email}"
     - Error: "Failed to sync {email}: {error}"
     - Continue to next user

Result: All changed users have updated PropertyValues
```

**Handling Missing Fields:**
```
User object: {
  "email": "john@example.com",
  "department": "Engineering"
  // Note: security_clearance is absent
}

Action: Only sync "department" field
        Do NOT set empty value for "security_clearance"
        User keeps existing value (if any) for absent fields
```

##### Step 5: Finalization

```
1. Calculate sync statistics:
   - Fields processed (created/updated)
   - Options added (per multiselect field)
   - Users processed (success/failed/skipped)
   - Total duration

2. Store sync metadata in KVStore:
   - Update last sync timestamp (for next incremental sync)
   - Update field mappings (if new fields created)
   - Update accumulated options (if new options added)
   - Store statistics for observability

3. Log comprehensive summary:
   INFO: Attribute sync completed
         Fields: 3 (2 existing, 1 created)
         Options added: 2 (to "security_clearance")
         Users: 15/17 synced (2 skipped - not found)
         Duration: 1.8s

4. Cluster job completes, scheduler will invoke again after interval
```

#### Error Handling Strategy

**Partial Failures (Continue Processing):**
- User email not found → Log warning, skip user, continue
- Field lookup fails → Log error, skip field for that user, continue
- Value upsert fails for one user → Log error, continue with next user

**Complete Failures (Abort Sync):**
- Provider initialization fails → Return error, retry on next interval
- Cannot fetch user data from provider → Return error, retry on next interval
- All field creations fail → Return error, investigate configuration

**Recovery:**
- Cluster job automatically reschedules after interval
- Idempotent operations (upserts, append-only options) allow safe retries
- No manual cleanup needed between runs

#### Design Constraints and Decisions

**1. No Field Type Changes**
- Once a field is created (text, multiselect, date), its type is immutable
- Rationale: Changing types would require deleting all user values
- Integrator responsibility: Ensure external API provides consistent types

**2. Append-Only Options (Multiselect)**
- Options are never removed, only added
- Rationale: Removing options could orphan user values
- Integrator responsibility: External API should not send users with invalid options

**3. No Value Deletion**
- If a field is absent from user object, we don't delete existing value
- Rationale: API sends only changed fields, absence != deletion
- Integrator responsibility: Explicitly set empty value to clear a field

**4. Email as User Identity**
- Users are matched by email address only
- Rationale: Email is the most reliable cross-system identifier
- Limitation: Users without emails cannot be synced

**5. Reserved Field Names**
- Only "email" is reserved and excluded from field creation
- Rationale: Keep it simple, avoid over-engineering
- Integrator responsibility: Avoid conflicts with Mattermost reserved keywords

### 4.5 Data Source Abstraction

A key design goal of this template is to make it easy to swap data sources. The plugin achieves this through the **AttributeProvider** interface pattern.

#### 4.5.1 AttributeProvider Interface

The AttributeProvider interface abstracts away the specifics of where and how user attribute data is retrieved.

**Interface Definition:**
```go
// AttributeProvider defines the contract for fetching user attribute data
// from external systems.
type AttributeProvider interface {
    // GetUserAttributes returns user objects that have changed.
    //
    // First call: Returns all users
    // Subsequent calls: Returns only users with changes since last call
    //
    // How "changed" is determined is an implementation detail of the provider.
    // Providers internally track state to support incremental updates.
    //
    // Returns:
    //   - Array of user attribute objects (as map[string]interface{})
    //   - Error if data retrieval fails
    GetUserAttributes() ([]map[string]interface{}, error)

    // Close cleans up any resources (connections, file handles, etc.)
    Close() error
}
```

**Contract and Expectations:**

1. **Incremental Updates:**
   - First call: Return all users
   - Subsequent calls: Return only changed users (new, modified, or deleted attributes)
   - Provider internally tracks what has been synced (implementation detail)
   - Empty array is valid response (no changes)

2. **User Object Format:**
   - Each user MUST have an "email" field (string)
   - All other fields become Custom Profile Attributes
   - Field values can be: string (text/date), array (multiselect)
   - Missing fields = don't update that attribute for that user

3. **Error Handling:**
   - Return error if data source is unavailable
   - Return error if data format is invalid
   - Plugin will retry on next sync interval

**Example User Object:**
```go
map[string]interface{}{
    "email": "john@example.com",
    "department": "Engineering",                    // string → text field
    "security_clearance": []string{"L2", "L3"},    // array → multiselect
    "start_date": "2023-01-15",                    // date string → date field
}
```

#### 4.5.2 File-Based Provider (Reference Implementation)

The template includes a file-based provider as the default implementation.

**Implementation Location:** `server/sync/file_provider.go`

**How It Works:**
```go
type FileProvider struct {
    filePath        string
    metadataPath    string
    lastReadTime    time.Time  // Internal state tracking
}

func (f *FileProvider) GetUserAttributes() ([]map[string]interface{}, error) {
    // 1. Read metadata to check if file has changed
    metadata, err := f.readMetadata()
    if err != nil {
        return nil, err
    }

    // 2. If file not modified since last read, return empty (no changes)
    if !f.lastReadTime.IsZero() && !metadata.LastModified.After(f.lastReadTime) {
        return []map[string]interface{}{}, nil
    }

    // 3. Read and parse JSON file
    data, err := os.ReadFile(f.filePath)
    if err != nil {
        return nil, err
    }

    var users []map[string]interface{}
    err = json.Unmarshal(data, &users)
    if err != nil {
        return nil, err
    }

    // 4. Update internal tracking
    f.lastReadTime = time.Now()

    return users, nil
}
```

**File Structure:**
```
assets/
├── user_attributes.json          # User data
└── user_attributes_metadata.json # Modification tracking
```

**Metadata Format:**
```json
{
  "last_modified": "2025-01-17T10:30:00Z",
  "user_count": 150,
  "description": "HR system export"
}
```

#### 4.5.3 Implementing a REST API Provider

Developers can easily create a REST API provider by implementing the interface.

**Example:**
```go
type RestAPIProvider struct {
    apiURL       string
    apiKey       string
    client       *http.Client
    lastSyncTime time.Time  // Internal state tracking
}

func (r *RestAPIProvider) GetUserAttributes() ([]map[string]interface{}, error) {
    // 1. Build API request - include lastSyncTime if not first call
    url := r.apiURL + "/users"
    if !r.lastSyncTime.IsZero() {
        url += "?since=" + r.lastSyncTime.Format(time.RFC3339)
    }

    // 2. Execute request with authentication
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+r.apiKey)

    resp, err := r.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // 3. Parse response
    var result struct {
        Users []map[string]interface{} `json:"users"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    // 4. Update internal tracking
    r.lastSyncTime = time.Now()

    return result.Users, nil
}
```

**Key Point:** The external API determines how to implement "changes since timestamp". The provider just passes that through.

#### 4.5.4 Implementing an LDAP Provider

**Example for LDAP integration:**
```go
type LDAPProvider struct {
    conn             *ldap.Conn
    baseDN           string
    lastModifyTime   string  // LDAP format timestamp
}

func (l *LDAPProvider) GetUserAttributes() ([]map[string]interface{}, error) {
    // 1. Build LDAP filter - only modified entries if not first sync
    filter := "(objectClass=person)"
    if l.lastModifyTime != "" {
        filter = fmt.Sprintf("(&(objectClass=person)(modifyTimestamp>=%s))",
            l.lastModifyTime)
    }

    // 2. Execute LDAP search
    searchRequest := ldap.NewSearchRequest(
        l.baseDN,
        ldap.ScopeWholeSubtree,
        ldap.NeverDerefAliases,
        0, 0, false,
        filter,
        []string{"mail", "department", "title"},
        nil,
    )

    sr, _ := l.conn.Search(searchRequest)

    // 3. Convert to user objects
    users := make([]map[string]interface{}, 0)
    for _, entry := range sr.Entries {
        users = append(users, map[string]interface{}{
            "email":      entry.GetAttributeValue("mail"),
            "department": entry.GetAttributeValue("department"),
            "title":      entry.GetAttributeValue("title"),
        })
    }

    // 4. Update tracking
    l.lastModifyTime = time.Now().Format("20060102150405Z")

    return users, nil
}
```

#### 4.5.5 Swapping Data Sources

To use a different data source, developers only need to:

1. **Implement the AttributeProvider interface** with their data source logic
2. **Update plugin initialization** to use the new provider
3. **Add configuration fields** for provider-specific settings (URLs, credentials, etc.)

**No changes required to:**
- Field synchronization logic
- Value synchronization logic
- Type inference
- Option management
- Cluster job scheduling

**Example Configuration:**
```go
func (p *Plugin) initializeProvider() (AttributeProvider, error) {
    config := p.getConfiguration()

    switch config.ProviderType {
    case "file":
        return NewFileProvider(config.FilePath), nil
    case "rest_api":
        return NewRestAPIProvider(config.APIURL, config.APIKey), nil
    case "ldap":
        return NewLDAPProvider(config.LDAPConfig), nil
    default:
        return nil, fmt.Errorf("unknown provider type: %s", config.ProviderType)
    }
}
```

#### 4.5.6 Design Benefits

**Clean Separation:**
- Core sync logic doesn't know about data source specifics
- Provider handles its own state tracking (timestamps, cursors, etc.)
- Easy to test with mock providers

**Flexibility:**
- Support any external system that can provide changed users
- External API determines "change" semantics (modified timestamp, version number, etc.)
- Plugin just processes whatever changed users are returned

**Example Mock for Testing:**
```go
type MockProvider struct {
    users []map[string]interface{}
    calls int
}

func (m *MockProvider) GetUserAttributes() ([]map[string]interface{}, error) {
    m.calls++
    if m.calls == 1 {
        return m.users, nil  // First call: all users
    }
    return []map[string]interface{}{}, nil  // Subsequent: no changes
}

func TestIncrementalSync(t *testing.T) {
    provider := &MockProvider{
        users: []map[string]interface{}{
            {"email": "test@example.com", "dept": "Eng"},
        },
    }

    // First sync processes users
    users1, _ := provider.GetUserAttributes()
    assert.Len(t, users1, 1)

    // Second sync gets no changes
    users2, _ := provider.GetUserAttributes()
    assert.Len(t, users2, 0)
}
```

## 5. Documentation Requirements

This section defines the documentation that must accompany the starter template to ensure developers can understand, customize, and deploy it successfully.

### 5.1 README.md

The README must provide a comprehensive overview and quick-start guide for developers. It should include an architecture overview, installation instructions, configuration guide, and links to detailed documentation. The README serves as the entry point for developers discovering the template.

### 5.2 Integration Guide

A detailed guide must be provided showing developers how to adapt the template for their specific external systems. This includes step-by-step instructions for implementing a custom AttributeProvider, adding authentication, customizing type inference logic, and handling system-specific data transformations. Example implementations for common scenarios (REST API, LDAP) should be included.

### 5.3 API Documentation

Inline code documentation (godoc) must explain all public interfaces, functions, and data structures. Complex logic and design decisions should be annotated with comments explaining the "why" behind implementation choices. The AttributeProvider interface and its contract must be thoroughly documented.

### 5.4 Example Data and Test Cases

The template must include well-commented example JSON data files demonstrating various scenarios: initial sync with all users, incremental sync with changes only, edge cases (special characters, empty values, new field types), and multiple users with different attribute combinations. Each example should explain what it demonstrates.

## Appendices

### Appendix A: Example Data Files

This appendix provides example data formats for the file-based AttributeProvider implementation.

#### A.1 User Attributes JSON Format

The file-based provider expects a JSON file with an array of user objects. Each object represents a user and their attributes.

**File Location:** `assets/user_attributes.json` or configurable path

**Schema:**
```json
[
  {
    "email": "string (required)",
    "field_name": "string | array | date_string",
    "another_field": "value",
    ...
  }
]
```

**Example - Initial Sync (All Users):**
```json
[
  {
    "email": "john.doe@example.com",
    "department": "Engineering",
    "location": "US-East",
    "security_clearance": ["Level2", "Level3"],
    "start_date": "2023-01-15",
    "employee_type": "Full-time"
  },
  {
    "email": "jane.smith@example.com",
    "department": "Sales",
    "location": "US-West",
    "security_clearance": ["Level1"],
    "start_date": "2022-08-01",
    "employee_type": "Full-time"
  },
  {
    "email": "bob.johnson@example.com",
    "department": "Marketing",
    "location": "EMEA",
    "security_clearance": ["Level1", "Level2"],
    "start_date": "2023-03-20",
    "employee_type": "Contractor"
  }
]
```

**Example - Incremental Sync (Changed Users Only):**
```json
[
  {
    "email": "john.doe@example.com",
    "department": "Engineering",
    "location": "US-West",
    "security_clearance": ["Level3", "Level4"]
  },
  {
    "email": "new.hire@example.com",
    "department": "Sales",
    "location": "APAC",
    "security_clearance": ["Level1"],
    "start_date": "2025-01-15",
    "employee_type": "Full-time"
  }
]
```

**Field Type Detection:**
- `"department": "Engineering"` → Text field
- `"security_clearance": ["Level1", "Level2"]` → Multiselect field
- `"start_date": "2023-01-15"` → Date field (ISO 8601 format)

**Important Notes:**
1. **email is required** - Every object must have an "email" field
2. **email is reserved** - Will not be created as a Custom Profile Attribute
3. **Missing fields are skipped** - If a user object doesn't have a field, that attribute won't be updated
4. **Empty arrays are valid** - `"security_clearance": []` clears multiselect value
5. **New fields auto-create** - Any new field names automatically create PropertyFields

#### A.2 Incremental Sync Metadata

The file-based provider tracks when data was last modified to support incremental sync.

**Metadata File Location:** `assets/user_attributes_metadata.json`

**Schema:**
```json
{
  "last_modified": "2025-01-17T10:30:00Z",
  "user_count": 150,
  "change_count": 15
}
```

**Usage:**
- Plugin passes last sync timestamp to provider
- Provider reads metadata to determine if file has changed
- If `last_modified` > last sync timestamp, reads and returns user data
- If no changes, returns empty array

#### A.3 Example Test Data

**Minimal Example (3 users, 3 fields):**
```json
[
  {
    "email": "alice@example.com",
    "team": "Frontend",
    "role": "Engineer"
  },
  {
    "email": "bob@example.com",
    "team": "Backend",
    "role": "Senior Engineer"
  },
  {
    "email": "carol@example.com",
    "team": "Design",
    "role": "Designer"
  }
]
```

**Complex Example (demonstrating all field types):**
```json
[
  {
    "email": "test.user@example.com",
    "single_text": "Simple text value",
    "date_field": "2023-06-15",
    "multiselect_tags": ["Tag1", "Tag2", "Tag3"],
    "empty_multiselect": [],
    "another_text": "More text"
  }
]
```

**Edge Cases Example:**
```json
[
  {
    "email": "edge.case@example.com",
    "special_chars": "Value with special: chars! @#$%",
    "long_text": "This is a very long text value that tests the character limit handling of text fields in Custom Profile Attributes",
    "unicode": "Unicode: 你好世界 🌍",
    "new_option": ["NewValue"]
  }
]
```

### Appendix B: Mattermost PropertyService API Reference

This appendix summarizes the key Mattermost PropertyService APIs used by the plugin.

#### B.1 Property Group Management

**Get or Register Property Group:**
```go
// Get the custom_profile_attributes group
group, err := pluginAPI.Property.GetPropertyGroup("custom_profile_attributes")

// If doesn't exist, register it
group, err := pluginAPI.Property.RegisterPropertyGroup("custom_profile_attributes")
```

**Returns:** `*model.PropertyGroup` with ID and Name

**Note:** Custom Profile Attributes always use the group name `"custom_profile_attributes"`. The group is automatically created by Mattermost core.

#### B.2 PropertyField Operations

**Create PropertyField:**
```go
field := &model.PropertyField{
    GroupID: groupID,
    Name:    "department",           // Internal field name
    Type:    model.PropertyFieldTypeText,
    Attrs: model.StringInterface{
        "display_name": "Department", // User-facing name
        "description":  "User's department",
        model.CustomProfileAttributesPropertyAttrsVisibility:
            model.CustomProfileAttributesVisibilityDefault,
    },
}

createdField, err := pluginAPI.Property.CreatePropertyField(field)
```

**Field Types:**
- `model.PropertyFieldTypeText` - Single-line text
- `model.PropertyFieldTypeSelect` - Single selection from options
- `model.PropertyFieldTypeMultiselect` - Multiple selections from options
- `model.PropertyFieldTypeDate` - Date value (ISO 8601)
- `model.PropertyFieldTypeUser` - Reference to Mattermost user
- `model.PropertyFieldTypeMultiuser` - References to multiple users

**Multiselect Field with Options:**
```go
field := &model.PropertyField{
    GroupID: groupID,
    Name:    "security_clearance",
    Type:    model.PropertyFieldTypeMultiselect,
    Attrs: model.StringInterface{
        "display_name": "Security Clearance",
        model.PropertyFieldAttributeOptions: []interface{}{
            map[string]interface{}{
                "id":   model.NewId(),
                "name": "Level1",
            },
            map[string]interface{}{
                "id":   model.NewId(),
                "name": "Level2",
            },
        },
    },
}
```

**Update PropertyField (for adding options):**
```go
updatedField, err := pluginAPI.Property.UpdatePropertyField(groupID, field)
```

**Get PropertyField:**
```go
field, err := pluginAPI.Property.GetPropertyField(groupID, fieldID)
```

**Search PropertyFields:**
```go
opts := model.PropertyFieldSearchOpts{
    GroupID: groupID,
    PerPage: 20,
}
fields, err := pluginAPI.Property.SearchPropertyFields(groupID, opts)
```

#### B.3 PropertyValue Operations

**Upsert Single PropertyValue:**
```go
value := &model.PropertyValue{
    GroupID:    groupID,
    TargetType: "user",
    TargetID:   userID,
    FieldID:    fieldID,
    Value:      json.RawMessage(`"Engineering"`), // JSON-encoded value
}

upsertedValue, err := pluginAPI.Property.UpsertPropertyValue(value)
```

**Upsert Multiple PropertyValues (Bulk):**
```go
values := []*model.PropertyValue{
    {
        GroupID:    groupID,
        TargetType: "user",
        TargetID:   userID1,
        FieldID:    fieldID1,
        Value:      json.RawMessage(`"Value1"`),
    },
    {
        GroupID:    groupID,
        TargetType: "user",
        TargetID:   userID1,
        FieldID:    fieldID2,
        Value:      json.RawMessage(`["opt1", "opt2"]`),
    },
}

upsertedValues, err := pluginAPI.Property.UpsertPropertyValues(values)
```

**Value Formats by Type:**

Text field:
```go
value.Value = json.RawMessage(`"Engineering"`)
```

Date field:
```go
value.Value = json.RawMessage(`"2023-01-15"`)
```

Multiselect field (option IDs):
```go
value.Value = json.RawMessage(`["opt_abc123", "opt_def456"]`)
```

Empty multiselect:
```go
value.Value = json.RawMessage(`[]`)
```

**Search PropertyValues:**
```go
opts := model.PropertyValueSearchOpts{
    GroupID:    groupID,
    TargetType: "user",
    TargetIDs:  []string{userID},
    PerPage:    100,
}
values, err := pluginAPI.Property.SearchPropertyValues(groupID, opts)
```

#### B.4 User Resolution

**Get User by Email:**
```go
user, appErr := pluginAPI.API.GetUserByEmail(email)
if appErr != nil {
    // User not found
}
userID := user.Id
```

#### B.5 Important Constraints

**Custom Profile Attributes Limits:**
- Maximum 20 fields per property group
- Field names must be unique within group
- Field types cannot be changed after creation
- Soft deletes only (DeleteAt timestamp)

**PropertyValue Constraints:**
- Values are JSON-encoded strings
- Multiselect values use option IDs, not option names
- Invalid option IDs in multiselect values are not validated by core (plugin must ensure validity)

### Appendix C: Learnings from PoC Implementation

This appendix documents key learnings from the `program-based-access-control` PoC plugin that informed this specification.

#### C.1 Cluster Job Pattern

**PoC Approach:**
The PoC initially used a goroutine with ticker for background jobs:
```go
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            p.runJob()
        case <-p.jobDone:
            return
        }
    }
}()
```

**Lesson Learned:**
Better to use Mattermost's cluster job system from the start. It provides:
- Automatic cluster awareness (only one instance runs)
- Leader election and failover
- No need for manual synchronization
- Cleaner lifecycle management

**Recommended Pattern:**
```go
job, err := cluster.Schedule(
    p.API,
    "AttributeSync",
    p.nextWaitInterval,
    p.runSync,
)
```

#### C.2 PropertyField Options Management

**Challenge:**
The PoC needed to add new program options dynamically while preserving existing option IDs.

**Solution Found:**
```go
// 1. Query existing field to get current options
existingField, _ := client.Property.GetPropertyField(groupID, fieldID)

// 2. Extract existing option name → ID mapping
existingOptions := make(map[string]string)
if opts, ok := existingField.Attrs[model.PropertyFieldAttributeOptions]; ok {
    // Parse options array
}

// 3. Build new options list, preserving IDs
newOptions := []interface{}{}
for _, programName := range programNames {
    optionID := existingOptions[programName]
    if optionID == "" {
        optionID = model.NewId() // Generate new ID only if needed
    }
    newOptions = append(newOptions, map[string]interface{}{
        "id":   optionID,
        "name": programName,
    })
}

// 4. Update field with merged options
patch := &model.PropertyFieldPatch{
    Attrs: &model.StringInterface{
        model.PropertyFieldAttributeOptions: newOptions,
    },
}
```

**Key Insight:** Always preserve existing option IDs to avoid breaking user values that reference those IDs.

#### C.3 Socket Client for Elevated Operations

**Discovery:**
Some operations require elevated privileges not available through standard Plugin API. The PoC used a socket client:
```go
socketClient := model.NewAPIv4SocketClient("/var/tmp/mattermost_local.socket")
```

**Use Cases:**
- Creating/updating Custom Profile Attribute fields via REST endpoints
- Operations that require admin permissions
- Bypassing plugin API limitations

**Consideration for Template:**
The starter template should document when socket client is needed vs regular Plugin API, but may not include it by default to keep the template simple.

#### C.4 User Attribute Synchronization Pattern

**Approach:**
```go
func (p *Plugin) SyncUserPrograms(person Person, accesses []Access) error {
    // 1. Resolve user by email
    mattermostID := person.MattermostID
    if mattermostID == "" && person.Email != "" {
        user, err := p.API.GetUserByEmail(person.Email)
        if err != nil {
            return err // Skip this user
        }
        mattermostID = user.Id
    }

    // 2. Determine attribute values from external data
    optionIDs := calculateValues(person, accesses)

    // 3. Upsert PropertyValue
    value := &model.PropertyValue{
        GroupID:    groupID,
        TargetType: "user",
        TargetID:   mattermostID,
        FieldID:    fieldID,
        Value:      json.Marshal(optionIDs),
    }
    _, err := p.client.Property.UpsertPropertyValue(value)
    return err
}
```

**Lessons:**
- Email resolution can fail - handle gracefully
- Use Upsert (not Create) to handle both new and existing values
- Batch operations when possible for performance

#### C.5 Error Handling Strategy

**Pattern from PoC:**
```go
for _, person := range persons {
    err := p.SyncUserPrograms(person, accesses)
    if err != nil {
        p.client.Log.Error("Failed to sync user",
            "person_id", person.ID,
            "email", person.Email,
            "error", err.Error())
        syncErrors = append(syncErrors, err)
        continue // Don't fail entire sync
    }
}

// Report summary
if len(syncErrors) > 0 {
    p.client.Log.Warn("Sync completed with errors",
        "success", len(persons) - len(syncErrors),
        "failed", len(syncErrors))
}
```

**Key Insight:** Partial failures should not stop entire sync. Log errors, track counts, continue processing.

#### C.6 Configuration-Driven Behavior

**PoC Configuration:**
```go
type configuration struct {
    UseDummyData   bool   `json:"use_dummy_data"`
    ExternalAPIURL string `json:"external_api_url"`
    ExternalAPIKey string `json:"external_api_key"`
}
```

**Pattern:**
- Support dummy/test data for development
- Make sync interval configurable
- Reload job when configuration changes

**Implementation:**
```go
func (p *Plugin) OnConfigurationChange() error {
    oldConfig := p.getConfiguration()
    newConfig := new(configuration)
    p.API.LoadPluginConfiguration(newConfig)

    if configChanged(oldConfig, newConfig) {
        // Restart sync job with new settings
        p.startSyncJob()
    }

    return nil
}
```

#### C.7 Performance Considerations

**PoC Observations:**
- Creating 100+ PropertyFields takes 5-10 seconds
- Upserting 1000 PropertyValues (bulk) takes 2-3 seconds
- User email lookup is relatively fast (< 100ms per user)

**Optimization Strategies:**
1. **Cache field mappings** in KVStore (don't query every sync)
2. **Batch PropertyValue upserts** (use UpsertPropertyValues not individual calls)
3. **Incremental sync** (only process changed users after first run)
4. **Minimize field updates** (only update options when new ones appear)

#### C.8 State Management with KVStore

**What to Store:**
```go
// Field name → Mattermost field ID mapping
key: "field_mapping_department"
value: "field_abc123xyz"

// Accumulated options per field
key: "field_options_security_clearance"
value: `{"Level1":"opt_123","Level2":"opt_456","Level3":"opt_789"}`

// Last successful sync timestamp
key: "last_sync_timestamp"
value: "2025-01-17T10:30:00Z"

// Sync statistics (optional)
key: "last_sync_stats"
value: `{"users_synced":150,"fields_created":5,"duration_seconds":3.2}`
```

**Key Insights:**
- KVStore is reliable and survives plugin restarts
- Serialize complex data as JSON strings
- Use namespaced keys to avoid conflicts

#### C.9 Logging Best Practices

**Structured Logging:**
```go
p.client.Log.Info("Attribute sync starting",
    "interval_minutes", intervalMinutes,
    "incremental", lastSyncTime != "",
)

p.client.Log.Debug("Processing user",
    "email", email,
    "field_count", len(attributes),
)

p.client.Log.Error("Failed to upsert values",
    "user_email", email,
    "error", err.Error(),
)
```

**Log Levels:**
- **DEBUG:** Per-user, per-field details
- **INFO:** Sync start/completion, summary statistics
- **WARN:** Recoverable issues (user not found, skipped)
- **ERROR:** Failures that need attention

#### C.10 What Not to Do

**Anti-Patterns Discovered:**

1. **Don't create fields on every sync** - Cache field IDs in KVStore
2. **Don't use goroutines for cluster jobs** - Use cluster.Schedule instead
3. **Don't fail entire sync on one error** - Continue with graceful degradation
4. **Don't update field options unnecessarily** - Check if new options exist first
5. **Don't sync all users every time** - Implement incremental sync
6. **Don't ignore email resolution failures** - Log and skip user, but continue
7. **Don't forget to handle plugin restart** - Ensure state persists in KVStore

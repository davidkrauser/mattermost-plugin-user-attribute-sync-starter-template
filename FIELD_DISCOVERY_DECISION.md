# How to Handle Field Discovery

## Context

We are building a Mattermost plugin starter template that synchronizes user profile attributes from external systems into Mattermost's Custom Profile Attributes. A critical design decision is how the plugin determines what fields to create.

This decision has significant impact on template complexity, developer experience, and flexibility.

## The Question

How should the plugin determine what Custom Profile Attribute fields to create and how should they be managed?

---

## Option A: Hardcoded Field Definitions

Developers define fields explicitly in code with known types and IDs. The schema is predetermined and known at compile time. All field IDs and option IDs are hardcoded as constants, eliminating the need for any ID lookup, storage, or caching mechanisms.

This approach treats the plugin code as the schema definition. When the plugin starts, it ensures the defined fields exist in Mattermost. During value synchronization, the code directly references the hardcoded field and option IDs without any runtime lookups.

**Example:**
```
// Define field and option IDs as constants
DepartmentFieldID = "dept_field_123"
ClearanceFieldID = "clear_field_789"
ClearanceLevel1ID = "opt_level1"
ClearanceLevel2ID = "opt_level2"

// Syncing user values - direct constant references
function syncUser(user):
    api.UpsertPropertyValues([
        {FieldID: DepartmentFieldID, Value: user.Department},
        {FieldID: ClearanceFieldID, Value: [ClearanceLevel1ID, ClearanceLevel2ID]},
    ])
```

**Key Point:** No runtime ID lookups, storage, or caching needed - all IDs are constants in code.

### Pros
- No ID management needed as Field IDs and option IDs are hardcoded constants, eliminating all ID mapping storage, caching, and lookup complexity
- Dramatically simpler (~80% less code) than dynamic creation
- No type inference logic
- Explicit and predictable - no surprises from data structure changes
- Easier for developers to understand and debug
- Better for production use (schema explicitly controlled)
- Field types and options clearly documented in code

### Cons
- Configuration burden - every integration requires editing field definitions
- Less adaptable - schema changes in external system require code changes
- Less educational - doesn't demonstrate as many field management patterns
- Different external systems need different field definitions
- Doesn't showcase Mattermost's dynamic field capabilities

---

## Option B: Dynamic Field Discovery

The plugin analyzes incoming user data to automatically discover what fields exist and what types they should be. When a new field appears in external data, the plugin creates a PropertyField in Mattermost automatically using type inference heuristics. Field IDs and option IDs are discovered at runtime and must be cached for performance.

This approach treats the external data structure as the schema definition. The plugin scans user objects to identify unique field names, infers appropriate types from the data patterns, and manages the lifecycle of fields and options automatically. All field and option IDs must be stored persistently and cached in memory to avoid expensive repeated lookups during value synchronization.

**Example:**
```
// Field discovery and creation
for each fieldName in external data:
    fieldID = CreatePropertyField(fieldName, inferredType)
    storage.Save(fieldName -> fieldID)
    cache.Save(fieldName -> fieldID)

// Multiselect option management
for each multiselect field:
    newOptions = DiscoverOptionsFromData()
    existingOptions = storage.GetOptions(fieldName)
    mergedOptions = Merge(existing, new)
    storage.SaveOptions(fieldName, mergedOptions)
    cache.SaveOptions(fieldName, mergedOptions)

// Value synchronization - requires ID lookups
function syncUser(user):
    for each (fieldName, value) in user:
        fieldID = cache.GetFieldID(fieldName)

        if isMultiselect(value):
            optionIDs = []
            for each optionName in value:
                optionID = cache.GetOptionID(fieldName, optionName)
                optionIDs.append(optionID)
            api.UpsertValue(fieldID, optionIDs)
        else:
            api.UpsertValue(fieldID, value)
```

**Key Point:** All field and option IDs must be persisted to storage and cached in memory for lookup during synchronization.

### Pros
- Zero configuration burden - works with any external data structure
- Adapts to schema evolution - new fields in external system automatically appear
- Works across diverse external systems without code changes
- Reduces integration time for developers
- Demonstrates PropertyService API patterns for educational value

### Cons
- Adds significant complexity
- Requires persistent storage for field name -> ID mappings
- Probably requires in-memory caching layer for performance
- Type inference heuristics can misclassify fields
- Multiselect option management complexity
- Harder to understand for developers learning the template
- Potential surprises if external data structure changes unexpectedly
- Performance overhead during field discovery phase

---

## Recommendation

**For the starter template: Option A (Hardcoded Field Definitions)**

**Rationale:**
- Easier for plugin writers to understand and extend
- Clear, explicit code that demonstrates the synchronization pattern without unnecessary complexity
- Developers can see exactly what fields are being created and how values are mapped
- Provides a solid foundation that developers can enhance as needed for their specific use case

**Important Note - SAF AQLX Project:**
While the starter template should use hardcoded fields for simplicity, the SAF AQLX project (which will be built on top of this template) will require some level of dynamic functionality. At minimum, **multiselect field options will need to be dynamically handled** to accommodate changing option values from the external system.

This means:
- The SAF AQLX implementation will need to extend the template with dynamic option management
- Field definitions themselves can remain hardcoded
- Option values for multiselect fields should be discovered and managed dynamically

---

## Request for Feedback

**Primary Question:** Do you agree with the recommendation to use hardcoded field definitions (Option A) for the starter template?

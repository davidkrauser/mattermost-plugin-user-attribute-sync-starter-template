# Claude Code Context

This document provides context for AI assistants working on the User Attribute Sync Starter Template plugin.

## Project Overview

This is a Mattermost plugin starter template designed to demonstrate how to synchronize user profile attributes from external systems into Mattermost's Custom Profile Attributes (CPA) system. It serves as both a working reference implementation and an educational resource for plugin developers.

**Key Characteristics:**
- Demonstrates dynamic field creation from JSON data structure
- Uses cluster-aware background jobs for reliable synchronization
- Implements incremental sync (only changed users after first run)
- Provides interface abstraction for swapping data sources
- Includes comprehensive documentation and examples

## Important Documents

### 1. Specification (`docs/SPECIFICATION.md`)

The complete technical specification for the plugin. This document defines:
- **Problem Statement**: Why this template exists and what challenges it solves
- **Requirements**: Functional and non-functional requirements with acceptance criteria
- **Architecture**: High-level design, component overview, data flow, and sync workflow
- **Design Constraints**: Key decisions and tradeoffs (field type immutability, append-only options, etc.)
- **Data Source Abstraction**: AttributeProvider interface design
- **Documentation Requirements**: What documentation must accompany the code
- **Appendices**: Example data formats, API references, and learnings from PoC implementation

**When to reference:**
- Understanding the "why" behind design decisions
- Clarifying requirements and acceptance criteria
- Understanding the data flow and architecture
- Looking up API usage patterns

### 2. Progress Tracking (`docs/PROGRESS.md`)

The detailed implementation plan broken into 22 phases. Each phase includes:
- Status tracking (Not Started / In Progress / Complete)
- Code changes with line estimates
- Unit test requirements
- Verification steps (`make test` and `make check-style`)
- Commit message guidance (explain WHY, not just WHAT)

**When to reference:**
- Starting work on a new phase
- Understanding what needs to be implemented
- Checking verification steps before committing
- Writing commit messages

## Implementation Workflow

**Each sub-phase is implemented as a separate commit.** When working on a sub-phase:

1. **Read the phase in PROGRESS.md** to understand scope and requirements
2. **Reference SPECIFICATION.md** for architectural context and design decisions
3. **Write the code** according to the phase description
4. **Write unit tests** for the code (same commit)
5. **Run verification:**
   ```bash
   make test
   make check-style
   ```
6. **Commit** with a detailed message explaining WHY:
   - What problem does this solve?
   - Why this approach was chosen?
   - Key design decisions made in the implementation
   - **IMPORTANT:** Do NOT reference SPECIFICATION.md or PROGRESS.md in commit messages
     - These documents may change or be removed in the future
     - Commit messages should stand alone as documentation of the change
   - Include Claude attribution:
     ```
     🤖 Generated with Claude Code

     Co-Authored-By: Claude <noreply@anthropic.com>
     ```

## Key Design Decisions

**Hardcoded Values (No Configuration):**
- Sync interval: 60 minutes
- File path: `data/user_attributes.json`
- Rationale: Keeps template simple; developers modify as needed for their use case

**No Metadata File:**
- Use `os.Stat()` to check file modification time directly
- Simpler than maintaining separate metadata file

**Cluster Jobs (Not Goroutines):**
- Use Mattermost's `cluster.Schedule()` for background work
- Provides automatic cluster awareness, leader election, and failover
- See spec Appendix C.1 for details

**Append-Only Multiselect Options:**
- Never remove options, only add new ones
- Prevents orphaning user values that reference removed options
- See spec FR4 and section 4.4 for details

**Email-Based User Identity:**
- Match external users to Mattermost users by email
- Most reliable cross-system identifier
- See spec FR5 for details

## Testing

**Unit Tests:**
- Each phase includes unit tests in `_test.go` files alongside code
- Use mocks for external dependencies (API, KVStore)
- Test happy path, error cases, and edge cases

**Integration Tests:**
- Phase 5.1 includes end-to-end integration test
- Validates all components work together
- Tests full sync flow with real file provider

**Verification Commands:**
```bash
make test           # Run all tests
make check-style    # Run linting
make clean          # Clean build artifacts
make all            # Run check-style, test, and build
```

## Common Pitfalls to Avoid

From spec Appendix C.10 (What Not to Do):
1. ❌ Don't create fields on every sync - cache field IDs in KVStore
2. ❌ Don't use goroutines for cluster jobs - use `cluster.Schedule` instead
3. ❌ Don't fail entire sync on one error - continue with graceful degradation
4. ❌ Don't update field options unnecessarily - check if new options exist first
5. ❌ Don't sync all users every time - implement incremental sync
6. ❌ Don't ignore email resolution failures - log and skip user, but continue
7. ❌ Don't forget to handle plugin restart - ensure state persists in KVStore

## Questions During Implementation?

If you encounter questions or need clarification:
1. **Check SPECIFICATION.md first** - most design decisions are documented
2. **Check PROGRESS.md** - the phase description may have more details
3. **Check spec appendices** - particularly Appendix C (PoC Learnings)
4. **Ask the user** - if something is ambiguous or requires a decision

## Current Status

See `docs/PROGRESS.md` for phase-by-phase status tracking.

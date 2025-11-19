package main

import (
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"

	"github.com/mattermost/user-attribute-sync-starter-template/server/store/kvstore"
	"github.com/mattermost/user-attribute-sync-starter-template/server/sync"
)

// syncIntervalMinutes defines how often the attribute sync job runs.
// This is hardcoded to keep the template simple - developers can modify this
// value directly based on their needs. For production use, consider making
// this configurable via plugin settings.
const syncIntervalMinutes = 60

// nextWaitInterval calculates the duration to wait before the next sync execution.
// This function is called by the cluster job scheduler to determine when to run
// the sync job next.
//
// On the first run (when metadata.LastFinished is zero), the job runs immediately.
// On subsequent runs, it waits for the configured interval from the last completion time.
//
// Why this approach:
// - Ensures consistent sync intervals regardless of how long sync takes
// - First run happens immediately on plugin activation for quick feedback
// - Uses cluster.JobMetadata to track execution history across restarts
func (p *Plugin) nextWaitInterval(now time.Time, metadata cluster.JobMetadata) time.Duration {
	// First run - execute immediately
	if metadata.LastFinished.IsZero() {
		return 0
	}

	// Calculate next scheduled run time
	nextRunTime := metadata.LastFinished.Add(syncIntervalMinutes * time.Minute)

	// If next run time is in the past, run immediately
	if nextRunTime.Before(now) {
		return 0
	}

	// Return duration until next scheduled run
	return nextRunTime.Sub(now)
}

// runSync executes the complete attribute synchronization workflow.
//
// This is the main orchestrator that coordinates:
//  1. Data fetching from external source via AttributeProvider
//  2. Field schema synchronization (create/update PropertyFields)
//  3. Value synchronization (upsert PropertyValues for all users)
//
// The function uses graceful degradation - field-level and user-level failures
// are logged but don't stop the entire sync. This ensures maximum progress even
// with partial data quality issues.
//
// Why this design:
//   - FieldCache loaded once and reused across both field and value sync
//   - Failed field creation doesn't prevent other fields from syncing
//   - Failed user value sync doesn't prevent other users from syncing
//   - All operations logged with context for debugging
//
// Error handling strategy:
//   - Provider initialization failure → abort (nothing to sync)
//   - FetchChanged failure → abort (no data to process)
//   - Field sync continues despite individual field failures
//   - Value sync continues despite individual user failures
func (p *Plugin) runSync() {
	p.client.Log.Info("Sync starting")

	// Initialize file provider
	fileProvider := sync.NewFileProvider()

	// Fetch changed users since last sync
	users, err := fileProvider.GetUserAttributes()
	if err != nil {
		p.client.Log.Error("Failed to fetch changed users", "error", err.Error())
		return
	}

	if len(users) == 0 {
		p.client.Log.Info("No changed users to sync")
		return
	}

	p.client.Log.Info("Fetched users for sync", "count", len(users))

	// Get or register Custom Profile Attributes group
	groupID, err := sync.GetOrRegisterCPAGroup(p.client)
	if err != nil {
		p.client.Log.Error("Failed to get CPA group", "error", err.Error())
		return
	}

	// Initialize FieldCache
	store := kvstore.NewKVStore(p.client)
	cache := sync.NewFieldCache(store)

	// Sync fields (creates/updates PropertyFields, populates cache)
	_, err = sync.SyncFields(p.client, groupID, users, cache)
	if err != nil {
		p.client.Log.Error("Failed to sync fields", "error", err.Error())
		return
	}

	// Sync user values (upserts PropertyValues using cache)
	err = sync.SyncUsers(p.client, groupID, users, cache)
	if err != nil {
		p.client.Log.Error("Failed to sync user values", "error", err.Error())
		return
	}

	p.client.Log.Info("Sync completed successfully", "users_processed", len(users))
}

package main

import (
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"

	"github.com/mattermost/user-attribute-sync-starter-template/server/sync"
)

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
// - Reads interval from plugin configuration to allow runtime customization
func (p *Plugin) nextWaitInterval(now time.Time, metadata cluster.JobMetadata) time.Duration {
	// Get the configured sync interval (defaults to 60 minutes if not set)
	config := p.getConfiguration()
	syncIntervalMinutes := config.SyncIntervalMinutes
	if syncIntervalMinutes < 1 {
		syncIntervalMinutes = 60 // Fallback to default if invalid
	}

	// First run - execute immediately
	if metadata.LastFinished.IsZero() {
		return 0
	}

	// Calculate next scheduled run time
	nextRunTime := metadata.LastFinished.Add(time.Duration(syncIntervalMinutes) * time.Minute)

	// If next run time is in the past, run immediately
	if nextRunTime.Before(now) {
		return 0
	}

	// Return duration until next scheduled run
	return nextRunTime.Sub(now)
}

// runSync executes the user attribute value synchronization workflow.
//
// This function runs periodically (at the interval configured in plugin settings) to synchronize
// user attribute values from external sources into Mattermost Custom Profile Attributes.
//
// Note: Field schema synchronization (creating/updating PropertyFields) happens
// once during plugin activation in OnActivate(). Since fields are hardcoded and
// unchanging, there's no need to sync them on every run.
//
// This orchestrator coordinates:
//  1. Data fetching from external source via AttributeProvider
//  2. Value synchronization (upsert PropertyValues for all users)
//
// The function uses graceful degradation - user-level failures are logged but
// don't stop the entire sync. This ensures maximum progress even with partial
// data quality issues.
//
// Error handling strategy:
//   - Provider initialization failure → abort (nothing to sync)
//   - Fetch failure → abort (no data to process)
//   - Value sync continues despite individual user failures
//   - All operations logged with context for debugging
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

	// Get Custom Profile Attributes group ID
	// The group and fields were already created during plugin activation
	groupID, err := sync.GetOrRegisterCPAGroup(p.client)
	if err != nil {
		p.client.Log.Error("Failed to get CPA group", "error", err.Error())
		return
	}

	// Sync user values (upserts PropertyValues using hardcoded field mappings)
	err = sync.SyncUsers(p.client, groupID, users)
	if err != nil {
		p.client.Log.Error("Failed to sync user values", "error", err.Error())
		return
	}

	p.client.Log.Info("Sync completed successfully", "users_processed", len(users))
}

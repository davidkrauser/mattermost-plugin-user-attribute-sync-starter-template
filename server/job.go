package main

import (
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
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

// runSync executes the attribute synchronization workflow.
// This is the main entry point for the cluster job and will be called
// periodically based on the interval defined by nextWaitInterval.
//
// In subsequent phases, this function will:
// - Initialize the AttributeProvider
// - Fetch changed users from external source
// - Sync fields (create/update PropertyFields)
// - Sync values (upsert PropertyValues for users)
// - Update last sync timestamp
//
// For now, it's a stub that logs execution to verify the cluster job is working.
func (p *Plugin) runSync() {
	p.client.Log.Info("Sync starting")
}

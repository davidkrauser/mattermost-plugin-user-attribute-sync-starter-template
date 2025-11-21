package main

import (
	"sync"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	attrsync "github.com/mattermost/user-attribute-sync-starter-template/server/sync"
	"github.com/pkg/errors"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// client is the Mattermost server API client.
	client *pluginapi.Client

	backgroundJob *cluster.Job

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin will be deactivated.
func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	// Sync hardcoded field definitions on plugin activation
	// Since fields are hardcoded and unchanging, we only need to create/update them
	// once when the plugin starts. This ensures fields exist and match our definitions.
	// If fields already exist, they'll be updated to match (idempotent operation).
	p.client.Log.Info("Syncing hardcoded field definitions on plugin activation")
	groupID, err := attrsync.GetOrRegisterCPAGroup(p.client)
	if err != nil {
		return errors.Wrap(err, "failed to get Custom Profile Attributes group")
	}

	err = attrsync.SyncFields(p.client, groupID)
	if err != nil {
		return errors.Wrap(err, "failed to sync hardcoded field definitions")
	}
	p.client.Log.Info("Field sync completed successfully")

	// Set up the attribute sync cluster job
	// This job runs periodically to synchronize user attribute values from external
	// sources to Mattermost Custom Profile Attributes. Using cluster.Schedule ensures
	// only one server instance runs the job in multi-server deployments (automatic
	// leader election and failover).
	job, err := cluster.Schedule(
		p.API,
		"AttributeSync",
		p.nextWaitInterval,
		p.runSync,
	)
	if err != nil {
		return errors.Wrap(err, "failed to schedule attribute sync job")
	}

	p.backgroundJob = job

	return nil
}

// OnDeactivate is invoked when the plugin is deactivated.
// Cleans up the attribute sync cluster job to prevent orphaned jobs.
func (p *Plugin) OnDeactivate() error {
	if p.backgroundJob != nil {
		if err := p.backgroundJob.Close(); err != nil {
			p.API.LogError("Failed to close attribute sync job", "err", err)
		}
	}
	return nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/

package main

import (
	"net/http"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/pluginapi/cluster"
	"github.com/mattermost/user-attribute-sync-starter-template/server/command"
	"github.com/mattermost/user-attribute-sync-starter-template/server/store/kvstore"
	attrsync "github.com/mattermost/user-attribute-sync-starter-template/server/sync"
	"github.com/pkg/errors"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// kvstore is the client used to read/write KV records for this plugin.
	kvstore kvstore.KVStore

	// client is the Mattermost server API client.
	client *pluginapi.Client

	// commandClient is the client used to register and execute slash commands.
	commandClient command.Command

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

	p.kvstore = kvstore.NewKVStore(p.client)

	p.commandClient = command.NewCommandHandler(p.client)

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

// This will execute the commands that were registered in the NewCommandHandler function.
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	response, err := p.commandClient.Handle(args)
	if err != nil {
		return nil, model.NewAppError("ExecuteCommand", "plugin.command.execute_command.app_error", nil, err.Error(), http.StatusInternalServerError)
	}
	return response, nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/

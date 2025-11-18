package kvstore

import (
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// Client exposes KVStore operations through a well-defined interface.
// This provides testability and stability by controlling how data is stored
// with specific keys and formats.
type Client struct {
	client *pluginapi.Client
}

// NewKVStore creates a new KVStore client wrapping the pluginapi.Client.
func NewKVStore(client *pluginapi.Client) KVStore {
	return Client{
		client: client,
	}
}

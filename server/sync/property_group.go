package sync

import (
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"
)

// customProfileAttributesGroupName is the standard group name for Custom Profile Attributes.
// This group is automatically created by Mattermost core and is used for all CPA fields.
//
//nolint:unused // Will be used in Phase 4.7
const customProfileAttributesGroupName = "custom_profile_attributes"

// getOrRegisterCPAGroup returns the ID of the Custom Profile Attributes property group.
// This group is required for all Custom Profile Attribute operations - fields must be
// associated with this group, and values must reference this group ID.
//
// The function first attempts to retrieve the existing group. If it doesn't exist
// (which should be rare, as Mattermost core typically creates it), the function
// attempts to register it.
//
// Why this helper is needed:
// All PropertyField and PropertyValue operations require the group ID. Rather than
// passing the group ID throughout the codebase, this helper provides a single point
// of access to retrieve it. This also handles the edge case where the group doesn't
// exist yet.
//
// Parameters:
//   - client: The pluginapi.Client used to access Mattermost Property APIs
//
// Returns:
//   - The Custom Profile Attributes group ID
//   - Error if the group cannot be retrieved or registered
func GetOrRegisterCPAGroup(client *pluginapi.Client) (string, error) {
	// First, try to get the existing group
	group, err := client.Property.GetPropertyGroup(customProfileAttributesGroupName)
	if err == nil && group != nil {
		return group.ID, nil
	}

	// If group doesn't exist, attempt to register it
	// Note: In normal Mattermost installations, this group should already exist
	group, err = client.Property.RegisterPropertyGroup(customProfileAttributesGroupName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get or register Custom Profile Attributes group")
	}

	if group == nil {
		return "", errors.New("RegisterPropertyGroup returned nil group")
	}

	return group.ID, nil
}

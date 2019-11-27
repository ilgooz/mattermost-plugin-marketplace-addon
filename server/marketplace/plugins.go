package marketplace

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

// Plugins is a list of Marketplace plugins.
type Plugins []*model.BaseMarketplacePlugin

// GetPlugin gets a plugin by id.
func (p *Plugins) GetPlugin(id string) (*model.BaseMarketplacePlugin, error) {
	for _, plugin := range *p {
		if plugin.Manifest.Id == id {
			return plugin, nil
		}
	}
	return nil, &NotFoundError{ID: id}
}

// NotFoundError error is returned when a plugin with id is not found.
type NotFoundError struct {
	// ID of the plugin.
	ID string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("plugin %q not found in the Marketplace", e.ID)
}

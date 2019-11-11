package marketplace

import (
	"github.com/mattermost/mattermost-server/model"
)

// Plugins is a list of Marketplace plugins.
type Plugins []*model.BaseMarketplacePlugin

// GetPlugin gets a plugin by id.
func (p *Plugins) GetPlugin(id string) (plugin *model.BaseMarketplacePlugin, found bool) {
	for _, plugin := range *p {
		if plugin.Manifest.Id == id {
			return plugin, true
		}
	}
	return nil, false
}

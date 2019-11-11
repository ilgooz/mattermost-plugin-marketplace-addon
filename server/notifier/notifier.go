package notifier

import (
	"fmt"

	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/updater"
	"github.com/mattermost/mattermost-server/plugin"
)

// Notifier notifies Mattermost admins and channels with plugin updates or failures.
type Notifier struct{}

func New(papi plugin.API, notifications chan updater.Notification, options ...Option) *Notifier {
	go func() {
		for a := range notifications {
			fmt.Println("notification", a.PluginID)
		}
	}()
	return &Notifier{}
}

func (n *Notifier) UpdateConfig(options ...Option) {}

type Option (*Notifier)

func NotificationChannelNameOption(channel string) Option { return nil }

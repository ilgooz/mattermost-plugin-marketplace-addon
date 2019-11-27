package notifier

import (
	"fmt"

	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/updater"
	"github.com/mattermost/mattermost-server/plugin"
)

// Notifier notifies Mattermost admins and channels with plugin updates or failures.
type Notifier struct{}

// New creates a new Notifier with papi and notifications chan.
// notifier consumes notifications from the notifications chan and sends notifications to Mattermost
// admins and to user given Mattermost notification #channel.
func New(papi plugin.API, notifications chan updater.Notification, options ...Option) *Notifier {
	go func() {
		for a := range notifications {
			fmt.Println("notification", a.PluginID)
		}
	}()
	return &Notifier{}
}

// UpdateConfig updates notifier's options configs set during the first initialization.
func (n *Notifier) UpdateConfig(options ...Option) {}

// Option modifies Notifier's configurations.
type Option (*Notifier)

// NotificationChannelNameOption creates a new option to set a Mattermost #channel to send
// notifications to.
func NotificationChannelNameOption(channel string) Option { return nil }

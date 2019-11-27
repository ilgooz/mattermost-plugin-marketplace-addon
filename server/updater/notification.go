package updater

// Notification is sent when a plugin is updated, cannot be updated or if any
// error occurred during the update process.
type Notification struct {
	// PluginID is the id of the plugin.
	PluginID string

	// Updated contains changelog information about the update and only filled
	// when a successful update is made.
	Updated *Changelog

	// Error can be a reason about why an update cannot be made, failed or can be
	// any other error.
	Error error
}

// Changelog contains information about the recent update.
type Changelog struct {
	// UpdatedName of the plugin.
	UpdatedName string

	// UpdatedDescription of the plugin.
	UpdatedDescription string

	// PreviousVersion of the plugin.
	PreviousVersion string

	// UpdatedVersion of the plugin.
	UpdatedVersion string
}

// notifyError sends an error notification.
func (u *Updater) notifyError(pluginID string, err error) {
	u.sendNotification(Notification{PluginID: pluginID, Error: err})
}

// notifyUpdated sends notification about a successful update.
func (u *Updater) notifyUpdated(pluginID string, changelog Changelog) {
	u.sendNotification(Notification{PluginID: pluginID, Updated: &changelog})
}

// sendNotification sends a notification to notification listener.
func (u *Updater) sendNotification(notification Notification) {
	conf := u.cloneConfing()
	if conf.notifications != nil {
		conf.notifications <- notification
	}
}

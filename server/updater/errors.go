package updater

import (
	"errors"
	"fmt"
)

var (
	// ErrNoNewerVersion error is returned when plugin has no a newer version on the Marketplace.
	ErrNoNewerVersion = errors.New("plugin has no a newer version")

	// ErrPluginInSkipList error is returned when plugin is in the skip plugins list.
	ErrPluginInSkipList = errors.New("plugin restricted to be installed, it is in the skip list")

	// ErrDifferentPlugins error is returned when two plugins are not the same by their id.
	ErrDifferentPlugins = errors.New("plugins are not the same, it cannot be updated")
)

// ServerVersionError is returned when new version of a plugin is not compatible
// with the current version of Mattermost Server.
type ServerVersionError struct {
	// PluginID of the Plugin.
	PluginID string

	// CurrentPluginVersion is the currently installed version of the plugin.
	CurrentPluginVersion string

	// NextPluginVersion is the newest version of the plugin that we tried to install.
	NextPluginVersion string

	// CurrentServerVersion is the current version of the server.
	CurrentServerVersion string

	// RequiredServerVersion is minimum required server version.
	RequiredServerVersion string
}

func (e *ServerVersionError) Error() string {
	return fmt.Sprintf("min required server version is %q to install %q version of %q plugin but server has a lower version %q",
		e.RequiredServerVersion, e.NextPluginVersion, e.PluginID, e.CurrentServerVersion)
}

package updater

import (
	"github.com/blang/semver"
	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/x/xstrings"
	"github.com/mattermost/mattermost-server/model"
)

// UpdateOp represents a new update operation between the installed plugin and the next plugin
// which is hosted in the Marketplace and might replace the installed one.
// UpdateOp used to compare these two plugins to find out if next plugin is really a new
// version of the installed one and is able to replace it.
type UpdateOp struct {
	// installed represents installed plugin.
	installed       *model.Manifest
	installedSemver semver.Version

	// next represent a plugin in the Marketplace. it might be a new version of
	// installed plugin or a totally different plugin.
	next       *model.BaseMarketplacePlugin
	nextSemver semver.Version

	// skipList used to skip updating a list of plugins by their ids.
	skipList []string

	// serverVersion is the Mattermost server's version.
	serverVersion string
}

// NewUpdateOp creates a new UpdateOp from installed and next plugin.
func NewUpdateOp(installed *model.Manifest, next *model.BaseMarketplacePlugin, skipList []string,
	serverVersion string) (*UpdateOp, error) {
	u := &UpdateOp{
		installed:     installed,
		next:          next,
		skipList:      skipList,
		serverVersion: serverVersion,
	}
	installedSemver, err := semver.Parse(installed.Version)
	if err != nil {
		return nil, err
	}
	nextSemver, err := semver.Parse(next.Manifest.Version)
	if err != nil {
		return nil, err
	}
	u.installedSemver = installedSemver
	u.nextSemver = nextSemver
	return u, nil
}

// CanBeUpdated compares installed plugin and next plugin to check if next plugin is
// able to replace the installed one as its newer version.
func (u *UpdateOp) CanBeUpdated() error {
	if err := u.requireNotSkipped(); err != nil {
		return err
	}
	if err := u.requireSamePlugin(); err != nil {
		return err
	}
	if err := u.requireNewerVersion(); err != nil {
		return err
	}
	return u.requireMinServerVersion()
}

// requireNotSkipped checks against if plugin is in the skip list.
func (u *UpdateOp) requireNotSkipped() error {
	if xstrings.SliceContains(u.skipList, u.installed.Id) {
		return ErrPluginInSkipList
	}
	return nil
}

// requireSamePlugin checks if installed and next plugins are the same plugins.
func (u *UpdateOp) requireSamePlugin() error {
	if u.installed.Id != u.next.Manifest.Id {
		return ErrDifferentPlugins
	}
	return nil
}

// requireNewerVersion checks if next plugin is a newer version of installed one.
func (u *UpdateOp) requireNewerVersion() error {
	if !u.nextSemver.GT(u.installedSemver) {
		return ErrNoNewerVersion
	}
	return nil
}

// requireMinServerVersion checks if the newer version of the plugin is compatible
// with the Mattermost server.
func (u *UpdateOp) requireMinServerVersion() error {
	if u.next.Manifest.MinServerVersion == "" {
		return nil
	}
	ok, err := u.next.Manifest.MeetMinServerVersion(u.serverVersion)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return &ServerVersionError{
		PluginID:              u.installed.Id,
		CurrentPluginVersion:  u.installed.Version,
		NextPluginVersion:     u.next.Manifest.Id,
		CurrentServerVersion:  u.serverVersion,
		RequiredServerVersion: u.next.Manifest.MinServerVersion,
	}
}

// CreateChangelog creates a changelog about plugin update by comparing the installed
// version with the next version.
func (u *UpdateOp) CreateChangelog() Changelog {
	return Changelog{
		PreviousVersion:    u.installed.Version,
		UpdatedName:        u.next.Manifest.Name,
		UpdatedDescription: u.next.Manifest.Description,
		UpdatedVersion:     u.next.Manifest.Version,
	}
}

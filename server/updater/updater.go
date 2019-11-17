// Package updater constantly updates plugins installed to the Mattermost Server to the latest versions.
// Check Option API to see how to configure an *Updater.
package updater

import (
	"context"
	"fmt"
	"sync"
	"time"

	dlock "github.com/ilgooz/mattermost-dlock"
	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/marketplace"
	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/x/xplugin"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

const (
	// defaultUpdateInterval used as a default wait time to wait before checking for updates again.
	defaultUpdateInterval = time.Minute * 3

	// updateLockKey used to do enable a distributed lock while doing update checks and updates.
	updateLockKey = "marketplace-addon:updater"
)

// Updater constantly updates plugins installed to the Mattermost Server to the latest versions.
type Updater struct {
	// papi use to access Mattermost API features.
	papi plugin.API

	// dlockStore used by the distributed lock to keep synchronization states.
	dlockStore dlock.Store

	// notifications chan forwards notifications related to plugin updates or failed update attempts.
	notifications chan Notification

	mc sync.RWMutex // protects config.
	// config holds configs set as options.
	conf *config

	// stopPooling stops poolling(checking for updates) -which means, it cancels Start().
	stopPooling context.CancelFunc
	// stopWait used to wait for stop proccess to be completed.
	stopWait *sync.WaitGroup
}

// Marketplace used to fetch latest versions of plugins from Mattermost Marketplace.
type Marketplace interface {
	ListPlugins() (marketplace.Plugins, error)
}

// config holds configs set as options.
type config struct {
	// marketplace used to fetch latest versions of plugins from Mattermost Marketplace.
	marketplace Marketplace

	// notifications chan forwards notifications related to plugin updates or failed update attempts.
	notifications chan Notification

	// updateInterval is used as a sleep time before checking for next updates.
	updateInterval time.Duration

	// skipPlugins is a list of plugins(ids) to be skipped during the update check.
	skipPlugins []string
}

// New creates new Updater with papi, marketplace, dlockStore and other options.
func New(papi plugin.API, marketplace Marketplace, dlockStore dlock.Store, options ...Option) *Updater {
	u := &Updater{
		papi:       papi,
		dlockStore: dlockStore,
		conf:       &config{marketplace: marketplace},
		stopWait:   &sync.WaitGroup{},
	}
	u.UpdateConfig(options...)
	// save notification chan at the beginning, so it cannot bu updated later by the UpdateConfig().
	u.notifications = u.conf.notifications
	return u
}

// UpdateConfig overwrites options that set during the first initialization of New.
func (u *Updater) UpdateConfig(options ...Option) {
	u.mc.Lock()
	defer u.mc.Unlock()
	for _, o := range options {
		o(u)
	}
	if u.conf.updateInterval == 0 {
		u.conf.updateInterval = defaultUpdateInterval
	}
}

// cloneConfing gets a snapshot of config's current state.
func (u *Updater) cloneConfing() config {
	u.mc.RLock()
	defer u.mc.RUnlock()
	return *u.conf
}

// Option used to customize Updater defaults.
type Option func(*Updater)

// UpdateIntervalOption sets a time to wait before checking for updates again.
func UpdateIntervalOption(interval time.Duration) Option {
	return func(u *Updater) {
		u.conf.updateInterval = interval
	}
}

// MarketplaceOption sets a marketplace where Updater can use to fetch Marketplace plugins.
func MarketplaceOption(marketplace Marketplace) Option {
	return func(u *Updater) {
		u.conf.marketplace = marketplace
	}
}

// SkipPluginsOption should provide a list of plugins(ids) to never update them.
func SkipPluginsOption(ids []string) Option {
	return func(u *Updater) {
		u.conf.skipPlugins = ids
	}
}

// NotificationsOption sets a notification chan to receive update related notifications.
// these notifications sent for every successful and unsuccessful updates and any errors
// occurred during an update process.
// notifications needs to be consumed within separate goroutines in order to not block the updater.
// notifications chan cannot be updated by the UpdateConfig(). trying it has no effects.
// notifications chan will be closed automatically after Updater is stopped.
func NotificationsOption(notifications chan Notification) Option {
	return func(u *Updater) {
		u.conf.notifications = notifications
	}
}

// Start starts updater to regularly check and do plugin updates.
// Start() blocks until pooling is cancelled and if there is any, waits for current update
// proccess to be competed before returning.
// calling Start() multiple times is a misuse. caller should initialize and use a new Updater.
func (u *Updater) Start() error {
	u.stopWait.Add(1)
	defer func() {
		defer u.stopWait.Done()
		if u.notifications != nil {
			close(u.notifications)
		}
	}()
	// create a cancelable context to stop pooling later.
	var ctx context.Context
	ctx, u.stopPooling = context.WithCancel(context.Background())
	// use a distributed lock here to only check and do updates in a single node(plugin instance)
	// at the same time.
	dl := dlock.New(updateLockKey, u.dlockStore)
	for {
		if err := dl.Lock(dlock.ContextOption(ctx)); err != nil {
			if err == context.Canceled {
				return nil
			}
			return err
		}
		// check and do updates.
		u.checkAndUpdate()
		// if there is a stop signal, it has the priority.
		select {
		case <-ctx.Done():
			dl.Unlock()
			return nil
		default:
		}
		conf := u.cloneConfing()
		afterC := time.After(conf.updateInterval)
		// wait for the next update round or a stop signal.
		select {
		// wait for updateInterval before doing the next check.
		// we start waiting after running the update check to make sure that once the Updater is
		// started, it will immediately start to the first update process instead of waiting.
		case <-afterC:
			dl.Unlock()
		case <-ctx.Done():
			dl.Unlock()
			return nil
		}
	}
}

// checkAndUpdate checks for new versions of installed plugins and updates them accordingly.
func (u *Updater) checkAndUpdate() {
	u.papi.LogInfo("checking for new versions...")
	updates := u.discover()
	lenUpdates := len(updates)
	if lenUpdates == 0 {
		u.papi.LogInfo("no new versions found")
		return
	}
	u.papi.LogInfo(fmt.Sprintf("found %d plugins to update", lenUpdates))
	var wg sync.WaitGroup
	wg.Add(lenUpdates)
	// TODO(ilgooz): limit the number of how many goroutines can be created.
	for _, updateOp := range updates {
		go func(updateOp *UpdateOp) {
			defer wg.Done()
			u.papi.LogInfo(fmt.Sprintf("updating %q from %q to %q...", updateOp.installed.Id,
				updateOp.installed.Version, updateOp.next.Manifest.Version))
			u.update(updateOp)
			u.papi.LogInfo(fmt.Sprintf("updated %q", updateOp.installed.Id))
		}(updateOp)
	}
	wg.Wait()
}

// discover discovers plugins that can be updated an returns a list of them.
func (u *Updater) discover() []*UpdateOp {
	var updates []*UpdateOp
	// get a list of installed plugins.
	installedPlugins, aerr := u.papi.GetPlugins()
	if aerr != nil {
		u.papi.LogInfo(errors.Wrap(aerr, "cannot get a list of installed plugins").Error())
		return nil
	}
	u.papi.LogInfo(fmt.Sprintf("found %d installed plugins", len(installedPlugins)))
	// if there are no installed plugins, there is nothing to update.
	if len(installedPlugins) == 0 {
		return nil
	}
	// get a list of Marketplace plugins.
	marketplacePlugins, err := u.conf.marketplace.ListPlugins()
	if err != nil {
		u.papi.LogError(errors.Wrap(err, "cannot get a list of plugins from Marketplace").Error())
		return nil
	}
	u.papi.LogInfo(fmt.Sprintf("found %d plugins in the marketplace", len(marketplacePlugins)))
	serverVersion := u.papi.GetServerVersion()
	// check every installed plugin to see if there is new versions.
	for _, manifest := range installedPlugins {
		// get the last version of the installed plugin from the Marketplace.
		// do nothing if the plugin is not in the Marketplace.
		marketplacePlugin, found := marketplacePlugins.GetPlugin(manifest.Id)
		if !found {
			continue
		}
		// create a new update operation for installed plugin and its version in the marketplace.
		updateOp, err := NewUpdateOp(manifest, marketplacePlugin, u.conf.skipPlugins, serverVersion)
		if err != nil {
			u.notifyError(manifest.Id, err)
			continue
		}
		// check if the plugin we get from the Marketplace is appropriate to replace the installed one.
		// if so add it to the updates list.
		if err := updateOp.CanBeUpdated(); err != nil {
			switch err {
			case ErrNoNewerVersion, ErrPluginInSkipList:
			default:
				u.notifyError(manifest.Id, err)
			}
			continue
		}
		updates = append(updates, updateOp)
	}
	return updates
}

// update updates an installed plugin by using info from updateOp.
func (u *Updater) update(updateOp *UpdateOp) {
	// install the plugin.
	_, aerr := xplugin.InstallPluginFromURL(u.papi, updateOp.next.DownloadURL, true)
	if aerr != nil {
		u.notifyError(updateOp.installed.Id, errors.Wrap(aerr, "could not install the plugin"))
		return
	}
	// create a changelog about the update.
	changelog := updateOp.CreateChangelog()
	// notify about the update.
	u.notifyUpdated(updateOp.installed.Id, changelog)
}

// Stop stops checking for updates and immediately returns.
// it does not interrupt current update process if there is any.
func (u *Updater) Stop() error {
	if u.stopPooling != nil {
		u.stopPooling()
		u.stopPooling = nil
	}
	return nil
}

// StopWait gracefull stops Updater by waiting current update process to be completed
// before returning.
// program can safely close after StopWait() returns.
func (u *Updater) StopWait() error {
	if err := u.Stop(); err != nil {
		return err
	}
	u.stopWait.Wait()
	return nil
}

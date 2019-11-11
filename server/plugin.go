package main

import (
	"time"

	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/marketplace"
	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/notifier"
	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/updater"
	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/x/xtime"
	"github.com/mattermost/mattermost-server/plugin"
)

// Plugin is Marketplace Addon that auto-updates plugins installed to Mattermost server.
type Plugin struct {
	plugin.MattermostPlugin

	// updater used to auto-update plugins.
	updater *updater.Updater
	// notifier used to send update notifications to admins and channels.
	notifier *notifier.Notifier

	// initialized keeps info about if all dependencies of this plugin are initialized or not.
	// initialization should be redone everytime plugin is activated.
	// initialized variable nor the initialization process is not protected with a mutex since it's assumed
	// that OnConfigurationChange(), OnActivate() and OnDeactivate() are called in order
	// and they are concurrency safe.
	initialized bool
}

// configuration holds Plugin's config.
// see plugin.json at the root dir for getting more info about these configurations.
type configuration struct {
	MarketplaceAPIAddress   string
	NotificationChannelName string
	UpdateCheckFrequency    xtime.Duration
}

func main() {
	plugin.ClientMain(&Plugin{})
}

// setup initializes and resets dependencies.
func (p *Plugin) setup() {
	notifications := make(chan updater.Notification)
	// no need to provide a marketplace instance here since it'll be done by OnConfigurationChange(),
	// and its called everytime when the configs are updated and at the first start time of the plugin.
	// we only do the initialization here with the constant configs.
	p.updater = updater.New(p.MattermostPlugin.API, nil, p.MattermostPlugin.API, []updater.Option{
		updater.NotificationsOption(notifications),
		updater.SkipPluginsOption([]string{manifest.ID}),
	}...)
	p.notifier = notifier.New(p.MattermostPlugin.API, notifications)
}

// start starts dependencies.
func (p *Plugin) start() {
	go func() {
		if err := p.updater.Start(); err != nil {
			p.logError(err)
		}
	}()
}

// stop stops dependencies. after calling stop(), a new call to setup() has to be made
// before making a start().
func (p *Plugin) stop(gracefullyWait bool) {
	if gracefullyWait {
		if err := p.updater.StopWait(); err != nil {
			p.logError(err)
		}
		p.logInfo("gracefully stopped")
		return
	}
	if err := p.updater.Stop(); err != nil {
		p.logError(err)
	}
}

// OnConfigurationChange setups and updates dependencies' configurations.
func (p *Plugin) OnConfigurationChange() error {
	var conf configuration
	if err := p.API.LoadPluginConfiguration(&conf); err != nil {
		return err
	}
	if !p.initialized {
		p.setup()
		p.initialized = true
	}
	p.updateConfig(conf)
	return nil
}

// updateConfig updates dependencies' configurations.
func (p *Plugin) updateConfig(conf configuration) {
	marketplace := marketplace.New(conf.MarketplaceAPIAddress)
	p.updater.UpdateConfig([]updater.Option{
		updater.MarketplaceOption(marketplace),
		updater.UpdateIntervalOption(time.Duration(conf.UpdateCheckFrequency)),
	}...)
	p.notifier.UpdateConfig([]notifier.Option{
		notifier.NotificationChannelNameOption(conf.NotificationChannelName),
	}...)
}

// OnActivate starts the plugin.
func (p *Plugin) OnActivate() error {
	defer p.logInfo("plugin is activated")
	p.start()
	return nil
}

// OnDeactivate stops the plugin.
func (p *Plugin) OnDeactivate() error {
	defer p.logInfo("plugin is deactivated")
	p.initialized = false
	p.stop(true)
	return nil
}

func (p *Plugin) logInfo(message string) {
	p.API.LogInfo(message)
}

func (p *Plugin) logError(err error) {
	p.API.LogError(err.Error())
}

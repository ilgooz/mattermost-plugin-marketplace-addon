package xplugin

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

// InstallPluginFromURL is Hard copied from the following PR, once the PR is merged, we'll use it and remove this func.
// Source: https://github.com/mattermost/mattermost-server/blob/f966aff56015fe2f7b9fda05a9715fb881503de9/plugin/helpers.go#L80
func InstallPluginFromURL(api plugin.API, downloadURL string, replace bool) (*model.Manifest, error) {
	client := &http.Client{Timeout: 60 * time.Minute}
	response, err := client.Get(downloadURL)
	if err != nil {
		return nil, errors.Wrap(err, "unable to download the plugin")
	}
	defer response.Body.Close()
	data, _ := ioutil.ReadAll(response.Body)
	manifest, appError := api.InstallPlugin(bytes.NewReader(data), true)
	if appError != nil {
		return nil, errors.Wrap(err, "unable to install plugin")
	}
	return manifest, nil
}

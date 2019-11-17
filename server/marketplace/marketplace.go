package marketplace

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"time"

	"github.com/mattermost/mattermost-server/model"
)

const (
	// listPluginsEndpoint is an endpoint to get plugin list.
	listPluginsEndpoint = "/api/v1/plugins"
)

const (
	// requestTimeout is used as a timeout value to cancel long running request made to Marketplace API.
	requestTimeout = time.Second * 10
)

// Marketplace is a gateway to interact with Mattermost Plugin Marketplace.
type Marketplace struct {
	// addr of the marketplace.
	addr string

	// client is used to perform HTTP requests to Marketplace server.
	client *http.Client
}

// New creates a new Marketplace with given Marketplace address addr.
func New(addr string) *Marketplace {
	return &Marketplace{
		addr:   addr,
		client: &http.Client{Timeout: requestTimeout},
	}
}

// ListPlugins fetches all plugins from the Marketplace.
func (m *Marketplace) ListPlugins() (Plugins, error) {
	urlParsed, err := url.Parse(m.addr)
	if err != nil {
		return nil, err
	}
	urlParsed.Path = listPluginsEndpoint
	resp, err := m.client.Get(urlParsed.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var data []byte
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(data))
	}
	plugins, err := model.BaseMarketplacePluginsFromReader(resp.Body)
	return plugins, err
}

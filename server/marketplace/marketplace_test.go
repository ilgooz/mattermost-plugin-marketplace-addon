package marketplace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/require"
)

func TestListPlugins(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/plugins", r.URL.Path)
		require.Equal(t, "GET", r.Method)
		data, _ := json.Marshal([]*model.BaseMarketplacePlugin{
			{Manifest: &model.Manifest{Id: "1"}},
			{Manifest: &model.Manifest{Id: "2"}},
		})
		w.Write(data)
	}))
	defer ts.Close()
	marketplace := New(ts.URL)
	plugins, err := marketplace.ListPlugins()
	require.NoError(t, err)
	require.Len(t, plugins, 2)
	require.Equal(t, "1", plugins[0].Manifest.Id)
	require.Equal(t, "2", plugins[1].Manifest.Id)
	plugin1, err := plugins.GetPlugin("1")
	require.NoError(t, err)
	require.Equal(t, "1", plugin1.Manifest.Id)
	plugin2, err := plugins.GetPlugin("2")
	require.NoError(t, err)
	require.Equal(t, "2", plugin2.Manifest.Id)
	_, err = plugins.GetPlugin("3")
	require.Equal(t, &NotFoundError{ID: "3"}, err)
}

func TestListPluginsBadStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("gone bad!"))
	}))
	defer ts.Close()
	marketplace := New(ts.URL)
	_, err := marketplace.ListPlugins()
	require.Equal(t, "gone bad!", err.Error())
}

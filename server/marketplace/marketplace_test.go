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
	plugin1, found1 := plugins.GetPlugin("1")
	plugin2, found2 := plugins.GetPlugin("2")
	_, found3 := plugins.GetPlugin("3")
	require.True(t, found1)
	require.Equal(t, "1", plugin1.Manifest.Id)
	require.True(t, found2)
	require.Equal(t, "2", plugin2.Manifest.Id)
	require.False(t, found3)
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

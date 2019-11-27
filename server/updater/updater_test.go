package updater

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/ilgooz/mattermost-dlock/dlocktest"
	"github.com/ilgooz/mattermost-plugin-marketplace-addon/server/marketplace"
	updatermock "github.com/ilgooz/mattermost-plugin-marketplace-addon/server/updater/mocks"
	apimock "github.com/ilgooz/mattermost-plugin-marketplace-addon/server/x/xplugin/mocks"
	"github.com/mattermost/mattermost-server/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TODO(ilgooz) add more tests.

func TestUpdate(t *testing.T) {
	ts := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer ts.Close()

	apiMock := &apimock.API{}
	apiMock.On("GetPlugins").Return([]*model.Manifest{
		{
			Id:      "topdf", // should update only topdf plugin.
			Version: "1.2.1",
		},
		{
			Id:      "github",
			Version: "2.3.0",
		},
	}, nil)
	apiMock.On("LogInfo", `checking for new versions...`).Once()
	apiMock.On("LogInfo", `found 2 plugins in the marketplace`).Once()
	apiMock.On("LogInfo", `found 2 installed plugins`).Once()
	apiMock.On("LogInfo", `found 1 plugins to update`).Once()
	apiMock.On("LogInfo", `updating "topdf" from "1.2.1" to "1.3.0"...`).Once()
	apiMock.On("LogInfo", `updated "topdf"`).Once()
	apiMock.On("LogError", (&marketplace.NotFoundError{ID: "github"}).Error()).Once()
	apiMock.On("GetServerVersion").Return("5.4.0")
	apiMock.On("InstallPlugin", mock.Anything, true).Once().Return(nil, nil).Run(func(args mock.Arguments) {
		tar := args.Get(0).(io.Reader)
		data, err := ioutil.ReadAll(tar)
		require.NoError(t, err)
		require.Equal(t, "topdf-0.1.3", string(data))
	})

	marketplaceMock := &updatermock.Marketplace{}
	marketplaceMock.On("ListPlugins").Return(marketplace.Plugins{
		{
			DownloadURL: buildDownloadURL(ts.URL, "topdf-0.1.3"),
			Manifest: &model.Manifest{
				Id:               "topdf",
				Version:          "1.3.0",
				MinServerVersion: "5.4.0",
				Name:             "TOPDF",
				Description:      "Create PDFs to preview Office files!",
			},
		},
		{
			DownloadURL: buildDownloadURL(ts.URL, "topdf-1.5.1"),
			Manifest: &model.Manifest{
				Id:               "antivirus",
				Version:          "1.5.1",
				MinServerVersion: "5.6.0",
				Name:             "Antivirus",
				Description:      "Scan attachments agains viruses!",
			},
		},
	}, nil)

	notifications := make(chan Notification)
	updater := New(apiMock, marketplaceMock, dlocktest.NewStore(), []Option{
		NotificationsOption(notifications),
		UpdateIntervalOption(time.Minute),
	}...)

	var startErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startErr = updater.Start()
	}()

	update := <-notifications
	require.NoError(t, update.Error)
	require.Equal(t, "topdf", update.PluginID)
	require.Equal(t, "TOPDF", update.Updated.UpdatedName)
	require.Equal(t, "Create PDFs to preview Office files!", update.Updated.UpdatedDescription)
	require.Equal(t, "1.3.0", update.Updated.UpdatedVersion)
	require.Equal(t, "1.2.1", update.Updated.PreviousVersion)

	select {
	case <-notifications:
		require.Fail(t, "there must be no further updates")
	default:
	}

	require.NoError(t, updater.Stop())
	wg.Wait()
	require.NoError(t, startErr)

	apiMock.AssertExpectations(t)
	marketplaceMock.AssertExpectations(t)
}

func buildDownloadURL(baseURL, file string) string {
	u, _ := url.Parse(baseURL)
	u.Path = path.Join(u.Path, file)
	return u.String()
}

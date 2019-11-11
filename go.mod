module github.com/ilgooz/mattermost-plugin-marketplace-addon

go 1.12

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/ilgooz/mattermost-dlock v0.0.0-20191107223609-c278c8c31ad6
	github.com/mattermost/mattermost-server v0.0.0-20191107143132-540cfb0239df
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.4.0
)

replace github.com/ilgooz/mattermost-dlock v0.0.0-20191107223609-c278c8c31ad6 => ../mattermost-dlock

replace github.com/mattermost/mattermost-server v0.0.0-20191107143132-540cfb0239df => ../../mattermost/mattermost-server

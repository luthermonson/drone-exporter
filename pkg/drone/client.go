package drone

import (
	"context"

	dronecli "github.com/drone/drone-go/drone"
	"github.com/jlehtimaki/drone-exporter/pkg/config"
	"golang.org/x/oauth2"
)

func GetClient(config *config.Config) dronecli.Client {
	oath := new(oauth2.Config)
	auth := oath.Client(
		context.TODO(),
		&oauth2.Token{
			AccessToken: config.Drone.Token,
		},
	)

	return dronecli.NewClient(config.Drone.Url, auth)
}

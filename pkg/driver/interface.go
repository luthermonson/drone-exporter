package driver

import (
	"github.com/jlehtimaki/drone-exporter/pkg/config"
	"github.com/jlehtimaki/drone-exporter/pkg/driver/influxdb"
	"github.com/jlehtimaki/drone-exporter/pkg/types"
)

const (
	INFLUXDB = "influxdb"
)

type Driver interface {
	Close() error
	LastBuildNumber(slug string) int64
	Batch(points []types.Point) error
}

func NewDriver(name string, config *config.Config) (Driver, error) {
	var d Driver
	var err error
	switch name {
	case INFLUXDB:
		d, err = influxdb.NewDriver(config)
	}

	return d, err
}

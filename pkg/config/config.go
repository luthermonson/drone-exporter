package config

import (
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

type (
	Config struct {
		Exporter
		InfluxDB
		Drone
		Repos []string
	}
	Exporter struct {
		Threads  int
		Interval int
		Driver   string
	}
	InfluxDB struct {
		Address  string
		Database string
		Username string
		Password string
	}
	Drone struct {
		Url   string
		Token string
	}
)

var Flags = []cli.Flag{
	&cli.IntFlag{
		Name:    "threads",
		Usage:   "How many repos to process at the same time",
		EnvVars: []string{"THREADS"},
		Value:   4,
	},
	&cli.IntFlag{
		Name:    "interval",
		Usage:   "How many repos to process at the same time",
		EnvVars: []string{"INTERVAL"},
		Value:   15,
	},
	&cli.StringFlag{
		Name:    "driver",
		Usage:   "Which backend driver to use, influxdb default and only supported",
		EnvVars: []string{"DRIVER"},
		Value:   "influxdb",
	},
	&cli.StringFlag{
		Name:    "influxdb-address",
		Usage:   "InfluxDB Address URL",
		EnvVars: []string{"INFLUXDB_ADDRESS"},
	},
	&cli.StringFlag{
		Name:    "influxdb-database",
		Usage:   "InfluxDB Database name",
		EnvVars: []string{"INFLUXDB_DATABASE"},
	},
	&cli.StringFlag{
		Name:    "influxdb-username",
		Usage:   "InfluxDB Username",
		EnvVars: []string{"INFLUXDB_USERNAME"},
	},
	&cli.StringFlag{
		Name:    "influxdb-password",
		Usage:   "InfluxDB Password",
		EnvVars: []string{"INFLUXDB_PASSWORD"},
	},
	&cli.StringFlag{
		Name:    "drone-url",
		Usage:   "Drone API URL",
		EnvVars: []string{"DRONE_URL"},
	},
	&cli.StringFlag{
		Name:    "drone-token",
		Usage:   "Drone API Token",
		EnvVars: []string{"DRONE_TOKEN"},
	},
}

func NewFromYaml(y string) (*Config, error) {
	var config Config
	err := yaml.Unmarshal([]byte(y), &config)
	return &config, err
}

func NewFromContext(c *cli.Context) *Config {
	return &Config{
		Exporter: Exporter{
			Threads:  c.Int("threads"),
			Interval: c.Int("interval"),
			Driver:   c.String("driver"),
		},
		InfluxDB: InfluxDB{
			Address:  c.String("influxdb-address"),
			Database: c.String("influxdb-database"),
			Username: c.String("influxdb-username"),
			Password: c.String("influxdb-password"),
		},
		Drone: Drone{
			Url:   c.String("drone-url"),
			Token: c.String("drone-token"),
		},
	}
}

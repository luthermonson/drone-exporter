package main

import (
	"io/ioutil"
	"os"

	"github.com/jlehtimaki/drone-exporter/pkg/config"
	"github.com/jlehtimaki/drone-exporter/pkg/exporter"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"github.com/urfave/cli/v2"
)

var version string

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "debug",
		Aliases: []string{"d"},
		Usage:   "Turn on verbose debug logging",
		EnvVars: []string{"LOG_LEVEL", "DEBUG"},
	},
	&cli.BoolFlag{
		Name:    "quiet",
		Aliases: []string{"q"},
		Usage:   "Turn on off all logging",
	},
	&cli.StringFlag{
		Name:     "config, c",
		Usage:    "config yaml file, see ./example/config.yml for structure",
		FilePath: "./config.yml",
	},
}

func main() {
	app := &cli.App{
		Name:    "drone-exporter",
		Usage:   "export drone data to a stats db",
		Action:  exporter.Run,
		Version: version,
		Flags:   append(flags, config.Flags...),
		Before: func(c *cli.Context) error {
			if c.Bool("quiet") {
				logrus.SetOutput(ioutil.Discard)
				return nil
			}

			debug := c.String("debug")
			if debug == "" {
				//treat logrus like fmt.Print
				logrus.SetFormatter(&easy.Formatter{
					LogFormat: "%msg%",
				})
				return nil
			}

			logLevel, err := logrus.ParseLevel(debug)
			if err != nil {
				// unparsed but intended for debug so default to DebugLevel
				logrus.SetLevel(logrus.DebugLevel)
			} else {
				logrus.SetLevel(logLevel)
			}
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}

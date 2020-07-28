package exporter

import (
	"sync"
	"time"

	dronecli "github.com/drone/drone-go/drone"
	"github.com/jlehtimaki/drone-exporter/pkg/config"
	"github.com/jlehtimaki/drone-exporter/pkg/driver"
	"github.com/jlehtimaki/drone-exporter/pkg/drone"
	"github.com/jlehtimaki/drone-exporter/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const pageSize = 25

type exporter struct {
	config *config.Config
	driver driver.Driver
	drone  dronecli.Client
}

func NewExporter(c *cli.Context) (*exporter, error) {
	var conf *config.Config
	var err error
	configYaml := c.String("config")

	if configYaml != "" {
		conf, err = config.NewFromYaml(c.String("config"))
		if err != nil {
			return nil, err
		}
	} else {
		conf = config.NewFromContext(c)
	}

	// initialize the influx client
	driver, err := driver.NewDriver(conf.Exporter.Driver, conf)
	if err != nil {
		return nil, err
	}

	return &exporter{
		config: conf,
		driver: driver,
		drone:  drone.GetClient(conf),
	}, nil
}

func Run(c *cli.Context) error {
	exporter, err := NewExporter(c)
	if err != nil {
		return err
	}
	defer exporter.driver.Close()

	var wg sync.WaitGroup
	sem := make(chan struct{}, exporter.config.Threads)

	// Start main loop
	for {

		repos, err := exporter.drone.RepoList()
		if err != nil {
			logrus.Fatal(err)
		}

		logrus.Infof("[drone-exporter] processing %d repos", len(repos))
		wg.Add(len(repos))
		for _, repo := range repos {
			r := repo
			go func() {
				logrus.Debugf("[%s] starting thread", r.Slug)
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				points := exporter.processRepo(r, exporter.driver.LastBuildNumber(r.Slug))
				if len(points) > 0 {
					logrus.Debugf("[%s] sending %d points to db", r.Slug, len(points))
					err = exporter.driver.Batch(points)
					if err != nil {
						logrus.Error(err)
					}
				}
				logrus.Debugf("[%s] thread complete", r.Slug)
			}()
		}
		wg.Wait()

		logrus.Infof("[drone-exporter] waiting %d minutes", exporter.config.Interval)
		time.Sleep(time.Duration(exporter.config.Interval) * time.Minute)
	}
}

func (e *exporter) processRepo(repo *dronecli.Repo, lastBuildId int64) []types.Point {
	var points []types.Point

	// process first page
	page := 1
	builds, err := e.drone.BuildList(repo.Namespace, repo.Name, dronecli.ListOptions{
		Page: page,
		Size: pageSize,
	})
	if err != nil {
		logrus.Fatal(err)
	}

	if len(builds) == 0 {
		logrus.Debugf("[%s] found zero builds, skipping...", repo.Name)
		return []types.Point{}
	}

	if builds[0].Number == lastBuildId {
		logrus.Debugf("[%s] found no new builds, skipping...", repo.Name)
		return []types.Point{}
	}

	points = append(points, e.processBuilds(repo, builds)...)
	if len(builds) < pageSize {
		return []types.Point{} //no pages
	}

	// paginate
	for len(builds) > 0 {
		page++
		builds, err = e.drone.BuildList(repo.Namespace, repo.Name, dronecli.ListOptions{
			Page: page,
			Size: pageSize,
		})
		if err != nil {
			logrus.Fatal(err)
		}

		if len(builds) == 0 {
			continue
		}
		ps := e.processBuilds(repo, builds)
		points = append(points, ps...)
		if len(builds) < pageSize {
			continue
		}
	}

	return points
}

func (e *exporter) processBuilds(repo *dronecli.Repo, builds []*dronecli.Build) []types.Point {
	logrus.Debugf("[%s] processing %d builds", repo.Slug, len(builds))
	var points []types.Point
	for _, build := range builds {
		buildInfo, err := e.drone.Build(repo.Namespace, repo.Name, int(build.Number))
		if err != nil {
			logrus.Fatal(err)
		}

		if buildInfo.Status == "running" {
			continue
		}

		var waittime int64
		if buildInfo.Started == 0 {
			waittime = buildInfo.Updated - buildInfo.Created
		} else {
			waittime = buildInfo.Started - buildInfo.Created
		}

		var duration int64
		if buildInfo.Finished == 0 {
			duration = buildInfo.Updated - buildInfo.Started
		} else {
			duration = buildInfo.Finished - buildInfo.Started
		}

		points = append(points, &types.Build{
			Time:     time.Unix(buildInfo.Started, 0),
			Number:   buildInfo.Number,
			Status:   buildInfo.Status,
			WaitTime: waittime,
			Duration: duration,
			Source:   buildInfo.Source,
			Target:   buildInfo.Target,
			Started:  buildInfo.Started,
			Created:  buildInfo.Created,
			Finished: buildInfo.Finished,
			BuildId:  build.Number,
			Tags: map[string]string{
				"DroneAddress": e.config.Drone.Url,
				"Slug":         repo.Slug,
				"Status":       buildInfo.Status,
			},
		})

		for _, stage := range buildInfo.Stages {
			// Loop through build info stages and save the results into DB
			// Don't save running pipelines and set BuildState integer according to the status because of Grafana
			var waittime int64
			if stage.Started == 0 {
				waittime = stage.Updated - stage.Created
			} else {
				waittime = stage.Started - stage.Created
			}

			var duration int64
			if stage.Stopped == 0 {
				duration = stage.Updated - stage.Started
			} else {
				duration = stage.Stopped - stage.Started
			}

			points = append(points, &types.Stage{
				Time:     time.Unix(stage.Started, 0),
				WaitTime: waittime,
				Duration: duration,
				OS:       stage.OS,
				Arch:     stage.Arch,
				Status:   stage.Status,
				Name:     stage.Name,
				BuildId:  build.Number,
				Tags: map[string]string{
					"DroneAddress": e.config.Drone.Url,
					"Slug":         repo.Slug,
					"Sender":       build.Sender,
					"Name":         stage.Name,
					"OS":           stage.OS,
					"Arch":         stage.Arch,
					"Status":       stage.Status,
				},
			})

			for _, step := range stage.Steps {
				duration := step.Stopped - step.Started
				if duration < 0 {
					duration = 0
				}
				points = append(points, &types.Step{
					Time:     time.Unix(step.Started, 0),
					Duration: duration,
					Name:     step.Name,
					Status:   step.Status,
					BuildId:  build.Number,
					Tags: map[string]string{
						"DroneAddress": e.config.Drone.Url,
						"Slug":         repo.Slug,
						"Sender":       build.Sender,
						"Name":         step.Name,
						"Status":       step.Status,
					},
				})
			}
		}
	}

	return points
}

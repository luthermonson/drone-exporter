package influxdb

import (
	"encoding/json"
	"fmt"
	"strconv"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/jlehtimaki/drone-exporter/pkg/config"
	"github.com/jlehtimaki/drone-exporter/pkg/types"
)

const LastBuildIdQueryFmt = `SELECT last("BuildId") AS "last_id" FROM "%s"."autogen"."builds" WHERE "Slug"='%s' AND "DroneAddress"='%s'`

type influxdb struct {
	client   client.Client
	address  string
	database string
}

func NewDriver(config *config.Config) (*influxdb, error) {
	client, err := getClient(config)
	if err != nil {
		return nil, err
	}
	return &influxdb{
		client:   client,
		database: config.InfluxDB.Database,
		address:  config.InfluxDB.Address,
	}, nil
}

func getClient(config *config.Config) (client.Client, error) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     config.InfluxDB.Address,
		Username: config.InfluxDB.Username,
		Password: config.InfluxDB.Password,
	})

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (d *influxdb) Close() error {
	return d.client.Close()
}

func (d *influxdb) LastBuildNumber(slug string) int64 {
	q := client.NewQuery(fmt.Sprintf(LastBuildIdQueryFmt, d.database, slug, d.address), d.database, "s")
	response, err := d.client.Query(q)
	if err != nil {
		return 0
	}

	if response.Error() != nil {
		return 0
	}

	if len(response.Results[0].Series) > 0 {
		s := string(response.Results[0].Series[0].Values[0][1].(json.Number))
		ret, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0
		}
		return ret
	}

	return 0
}

func (d *influxdb) Batch(points []types.Point) error {
	// Create a new point batch
	var bp client.BatchPoints
	var err error

	bp, err = client.NewBatchPoints(client.BatchPointsConfig{
		Database:  d.database,
		Precision: "s",
	})
	if err != nil {
		return err
	}

	i := 0
	for _, point := range points {

		pt, err := client.NewPoint(point.GetMeasurement(), point.GetTags(), point.GetFields(), point.GetTime())
		if err != nil {
			return err
		}
		bp.AddPoint(pt)

		i++

		// max batch of 10k
		if i > 500 {
			i = 0
			if err := d.client.Write(bp); err != nil {
				return err
			}
			bp, err = client.NewBatchPoints(client.BatchPointsConfig{
				Database:  d.database,
				Precision: "s",
			})
			if err != nil {
				return err
			}
		}
	}

	if err := d.client.Write(bp); err != nil {
		return err
	}

	return nil
}

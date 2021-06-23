package influx

import (
	client "github.com/influxdata/influxdb1-client/v2"
	"time"
)

type Influx struct {
	database string
	client   client.Client
}

func Open(address string, database string) (*Influx, error) {
	influxClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: address,
	})

	_, _, err = influxClient.Ping(0)

	return &Influx{
		database: database,
		client:   influxClient,
	}, err
}

func (influx *Influx) NewBatchPoints() (client.BatchPoints, error) {
	return client.NewBatchPoints(client.BatchPointsConfig{
		Precision: "us",
		Database:  influx.database,
	})
}

func (influx *Influx) Write(bp client.BatchPoints) error {
	return influx.client.Write(bp)
}

func (influx *Influx) Query(query string) (*client.Response, error) {
	q := client.NewQuery(query, influx.database, "")

	return influx.client.Query(q)
}

func (influx *Influx) Exec(tags map[string]string, fields map[string]interface{}, time time.Time) error {
	batchPoint, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  influx.database,
		Precision: "us",
	})

	point, err := client.NewPoint(
		"cloudData",
		tags,
		fields,
		time,
	)
	if err != nil {
		return err
	}

	batchPoint.AddPoint(point)

	return influx.client.Write(batchPoint)
}

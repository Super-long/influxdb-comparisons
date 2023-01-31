package influxdb

import (
	"fmt"
	"time"

	bulkQuerygen "github.com/influxdata/influxdb-comparisons/bulk_query_gen"
)

type InfluxMetaquery struct {
	InfluxCommon
}

// queryInterval is currently not used, but may be used to include as a parameter in the future.
func NewInfluxMetaqueryCommon(lang Language, dbConfig bulkQuerygen.DatabaseConfig, queriesFullRange bulkQuerygen.TimeInterval, queryInterval time.Duration, scaleVar int) bulkQuerygen.QueryGenerator {
	if _, ok := dbConfig[bulkQuerygen.DatabaseName]; !ok {
		panic("need influx database name")
	}

	return &InfluxMetaquery{
		InfluxCommon: *newInfluxCommon(lang, dbConfig[bulkQuerygen.DatabaseName], dbConfig[bulkQuerygen.UserName], dbConfig[bulkQuerygen.Password], queriesFullRange, scaleVar),
	}
}

// Dispatch is to satisfy the bulkQuerygen.QueryGenerator interface. Specific
// query-types will implement their own Dispatch methods which will supercede
// this one. Returning the query without setting it for this root method follows
// the convention in influx_dashboard_common.go
func (d *InfluxMetaquery) Dispatch(i int) bulkQuerygen.Query {
	q := bulkQuerygen.NewHTTPQuery() // from pool
	return q
}

// MetaqueryTagValues generates a query that returns a list of tag values for a
// specific tag key. The InfluxQL query is very simple and is constant. The Flux
// query is slightly more complicated and is intended to replicate an equivalent
// query that would be generated by the InfluxDB UI for getting "all" the tag
// values for a specific tag key.
func (d *InfluxMetaquery) MetaqueryTagValues(qi bulkQuerygen.Query) {
	var query string
	if d.language == InfluxQL {
		query = `SHOW TAG VALUES FROM "example_measurement" WITH KEY = "X" LIMIT 200`
	} else {
		query = fmt.Sprintf(`from(bucket: "%s") `+
			`|> range(start: %s, stop: %s) `+
			`|> filter(fn: (r) => (r["_measurement"] == "example_measurement"))`+
			`|> keep(columns: ["X"])`+
			`|> group()`+
			`|> distinct(column: "X")`+
			`|> limit(n: 200)`+
			`|> sort()`,
			d.DatabaseName,
			d.AllInterval.StartString(),
			d.AllInterval.EndString(),
		)
	}

	humanLabel := fmt.Sprintf(`InfluxDB (%s) tag values for KEY = "X"`, d.language)
	q := qi.(*bulkQuerygen.HTTPQuery)
	d.getHttpQuery(humanLabel, "n/a", query, q)
}

// MetaqueryFieldKeys generates a query that returns a list of field keys for a
// specific measurement.
func (d *InfluxMetaquery) MetaqueryFieldKeys(qi bulkQuerygen.Query) {
	var query string
	if d.language == InfluxQL {
		query = `SHOW FIELD KEYS FROM "example_measurement" LIMIT 200`
	} else {
		query = fmt.Sprintf(`from(bucket: "%s") `+
			`|> range(start: %s, stop: %s) `+
			`|> filter(fn: (r) => (r["_measurement"] == "example_measurement"))`+
			`|> keep(columns: ["_field"])`+
			`|> group()`+
			`|> distinct(column: "_field")`+
			`|> limit(n: 200)`+
			`|> sort()`,
			d.DatabaseName,
			d.AllInterval.StartString(),
			d.AllInterval.EndString(),
		)
	}

	humanLabel := fmt.Sprintf(`InfluxDB (%s) field keys`, d.language)
	q := qi.(*bulkQuerygen.HTTPQuery)
	d.getHttpQuery(humanLabel, "n/a", query, q)
}

// MetaqueryCardinality calculates the series cardinality for all data in a
// bucket. The Flux query uses an arbitrarily large time range to ensure that
// all shards are within that time range, in order to be comparable to InfluxQL
// which does not use a time range for cardinality estimation.
func (d *InfluxMetaquery) MetaqueryCardinality(qi bulkQuerygen.Query) {
	var query string
	if d.language == InfluxQL {
		query = fmt.Sprintf(`SHOW SERIES EXACT CARDINALITY ON %s`, d.DatabaseName)
	} else {
		query = fmt.Sprintf(`import "influxdata/influxdb"

		influxdb.cardinality(
			bucket: "%s",
			start: -100y,
			stop: 2030-01-01T00:00:00Z,
		)`,
			d.DatabaseName,
		)
	}

	humanLabel := fmt.Sprintf(`InfluxDB (%s) Series Cardinality`, d.language)
	q := qi.(*bulkQuerygen.HTTPQuery)
	d.getHttpQuery(humanLabel, "n/a", query, q)
}

package metricscollector

import (
	"cmp"
	"slices"

	"hump/internal/domain"
	port "hump/internal/port/metrics_collector"
)

// sinkEndpoint identifies one sink connection, the level the targets of a
// snapshot are grouped under.
type sinkEndpoint struct {
	sinkType string
	host     string
}

// snapshot renders the state of the collector for d. An idle collector has no
// readings left, so it reports its topology with zero counts rather than
// nothing at all. c.mu must be held.
func (c *Collector) snapshot(d port.Duration) port.Snapshot {
	var readings latest
	if c.scraper != nil {
		readings = c.scraper.latest
	}

	points := []port.Point{}
	var windowed map[sinkKey]*histogram

	if s, ok := c.series[d]; ok {
		points = s.dump()
		windowed = s.total().sinkLatency
	}

	return port.Snapshot{
		Points:      points,
		Tables:      c.tables(readings),
		Sinks:       c.sinks(readings, windowed),
		Connections: c.edges(),
	}
}

// tables counts the row operations of every configured table, keyed by
// "schema.table".
func (c *Collector) tables(readings latest) map[string]port.TableOps {
	tables := make(map[string]port.TableOps, len(c.topology))
	for _, conn := range c.topology {
		tables[qualify(conn.Table)] = readings.tables[conn.Table]
	}

	return tables
}

// sinks groups the configured topology by sink endpoint, reporting the counts of
// each endpoint and of every target it renders to.
func (c *Collector) sinks(readings latest, windowed map[sinkKey]*histogram) []port.Sink {
	order := make([]sinkEndpoint, 0, len(c.topology))
	targets := make(map[sinkEndpoint][]string, len(c.topology))

	for _, conn := range c.topology {
		endpoint := sinkEndpoint{sinkType: conn.SinkType, host: conn.Host}
		if _, ok := targets[endpoint]; !ok {
			order = append(order, endpoint)
			targets[endpoint] = []string{}
		}

		if !slices.Contains(targets[endpoint], conn.Target) {
			targets[endpoint] = append(targets[endpoint], conn.Target)
		}
	}

	slices.SortFunc(order, func(a, b sinkEndpoint) int {
		return cmp.Or(cmp.Compare(a.sinkType, b.sinkType), cmp.Compare(a.host, b.host))
	})

	sinks := make([]port.Sink, 0, len(order))
	for _, endpoint := range order {
		names := targets[endpoint]
		slices.Sort(names)

		keys := make([]sinkKey, 0, len(names))
		rendered := make([]port.Target, 0, len(names))

		for _, name := range names {
			key := sinkKey{sinkType: endpoint.sinkType, host: endpoint.host, target: name}
			keys = append(keys, key)
			rendered = append(rendered, port.Target{
				Target:   name,
				Delivery: delivery(readings, windowed, []sinkKey{key}),
			})
		}

		sinks = append(sinks, port.Sink{
			Type:     endpoint.sinkType,
			Host:     endpoint.host,
			Targets:  rendered,
			Delivery: delivery(readings, windowed, keys),
		})
	}

	return sinks
}

// edges counts the deliveries reported over every configured connection. The
// counts are pushed in by the pipeline through Record rather than scraped, since
// the sink metrics carry no table.
func (c *Collector) edges() []port.ConnectionOps {
	c.recordedMu.Lock()
	defer c.recordedMu.Unlock()

	conns := make([]port.ConnectionOps, 0, len(c.topology))
	for _, conn := range c.topology {
		byStatus := c.recorded[conn]
		conns = append(conns, port.ConnectionOps{
			Connection: conn,
			Operations: byStatus[domain.DeliveryStatusSuccess] + byStatus[domain.DeliveryStatusFailure],
		})
	}

	slices.SortFunc(conns, func(a, b port.ConnectionOps) int {
		return cmp.Or(
			cmp.Compare(a.Table.Schema, b.Table.Schema),
			cmp.Compare(a.Table.Name, b.Table.Name),
			cmp.Compare(a.SinkType, b.SinkType),
			cmp.Compare(a.Host, b.Host),
			cmp.Compare(a.Target, b.Target),
		)
	})

	return conns
}

// delivery sums the lifetime counts of keys and estimates their p95 latency over
// the scraped window, in milliseconds.
func delivery(readings latest, windowed map[sinkKey]*histogram, keys []sinkKey) port.Delivery {
	var success, failure float64
	merged := histogram{buckets: make(map[float64]float64)}

	for _, key := range keys {
		byStatus := readings.sinks[key]
		success += byStatus[domain.DeliveryStatusSuccess]
		failure += byStatus[domain.DeliveryStatusFailure]

		if h, ok := windowed[key]; ok {
			merged.add(*h)
		}
	}

	result := port.Delivery{Operations: success + failure}

	if result.Operations > 0 {
		rate := success / result.Operations
		result.Success = &rate
	}

	if value, ok := merged.quantile(latencyQuantile); ok {
		milliseconds := value * millisecondsPerSecond
		result.P95 = &milliseconds
	}

	return result
}

// qualify renders a table identifier as the schema-qualified name a snapshot
// keys its tables by.
func qualify(table domain.TableIdentifier) string {
	return string(table.Schema) + "." + string(table.Name)
}

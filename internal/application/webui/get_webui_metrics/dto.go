package getwebuimetrics

import "time"

type MetricsRequest struct {
	Body struct {
		Duration string `json:"duration" enum:"1m,5m,10m" doc:"Width of the buckets the points are aggregated into"`
	}
}

type MetricsPoint struct {
	Timestamp           time.Time          `json:"timestamp"           doc:"Start of the bucket"`
	Delivered           int64              `json:"delivered"           doc:"Messages delivered successfully during the bucket"`
	Failed              int64              `json:"failed"              doc:"Messages that failed to deliver during the bucket"`
	Throughput          float64            `json:"throughput"          doc:"Messages delivered successfully per second"`
	ReplicationLagBytes float64            `json:"replicationLagBytes" doc:"Peak replication lag in bytes"`
	MemoryAllocBytes    float64            `json:"memoryAllocBytes"    doc:"Peak allocated heap in bytes"`
	SinkDelivery        map[string]int64   `json:"sinkDelivery"        doc:"Delivered and failed messages by sink type"`
	SinkLatencyP95      map[string]float64 `json:"sinkLatencyP95"      doc:"P95 sink write latency in milliseconds by sink type; a type without an estimate is absent"`
}

// MetricsTableOps abbreviates its keys because the map that carries it repeats
// them for every configured table.
type MetricsTableOps struct {
	Insert   int64 `json:"i" doc:"Rows inserted since start"`
	Update   int64 `json:"u" doc:"Rows updated since start"`
	Delete   int64 `json:"d" doc:"Rows deleted since start"`
	Truncate int64 `json:"t" doc:"Tables truncated since start"`
}

type MetricsSinkTarget struct {
	Target     string   `json:"target"            doc:"Rendered target: topic, subject, exchange, stream or HTTP method"`
	Operations int64    `json:"operations"        doc:"Deliveries finished since start, retries excluded"`
	Success    *float64 `json:"success,omitempty" doc:"Share of deliveries that succeeded, 0 to 1; absent without an operation"`
	P95        *float64 `json:"p95,omitempty"     doc:"P95 latency of a single write attempt in milliseconds over the requested window; absent without an estimate"`
}

type MetricsSink struct {
	Type       string              `json:"type"              doc:"Sink type"`
	Host       string              `json:"host"              doc:"Connection host the sink URL renders to"`
	Operations int64               `json:"operations"        doc:"Deliveries finished since start, retries excluded"`
	Success    *float64            `json:"success,omitempty" doc:"Share of deliveries that succeeded, 0 to 1; absent without an operation"`
	P95        *float64            `json:"p95,omitempty"     doc:"P95 latency of a single write attempt in milliseconds over the requested window; absent without an estimate"`
	Targets    []MetricsSinkTarget `json:"targets"           doc:"Distinct targets the sink renders to; tables sharing a target collapse into one entry"`
}

type MetricsConnection struct {
	Table      string `json:"table"      doc:"Schema-qualified source table"`
	Sink       string `json:"sink"       doc:"Sink type"`
	Host       string `json:"host"       doc:"Connection host the sink URL renders to"`
	Target     string `json:"target"     doc:"Rendered target the table is delivered to"`
	Operations int64  `json:"operations" doc:"Deliveries finished over the connection since start, retries excluded"`
}

type MetricsResponse struct {
	Body struct {
		Duration      string                     `json:"duration"      doc:"Width of the buckets the points are aggregated into"`
		BucketSeconds int                        `json:"bucketSeconds" doc:"Bucket width in seconds"`
		Points        []MetricsPoint             `json:"points"        doc:"Newest points, oldest first"`
		Tables        map[string]MetricsTableOps `json:"tables"        doc:"Row operations by configured table, keyed by schema-qualified name"`
		Sinks         []MetricsSink              `json:"sinks"         doc:"Configured sinks, with the targets they render to"`
		Connections   []MetricsConnection        `json:"connections"   doc:"Configured table-to-target edges of the pipeline"`
	}
}

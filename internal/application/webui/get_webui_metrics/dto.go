package getwebuimetrics

import "time"

type MetricsRequest struct {
	Body struct {
		Duration string `json:"duration" enum:"1m,5m,10m" doc:"Width of the buckets the points are aggregated into"`
	}
}

type MetricsPoint struct {
	Timestamp             time.Time          `json:"timestamp"             doc:"Start of the bucket"`
	Delivered             int64              `json:"delivered"             doc:"Messages delivered successfully during the bucket"`
	Failed                int64              `json:"failed"                doc:"Messages that failed to deliver during the bucket"`
	Throughput            float64            `json:"throughput"            doc:"Messages delivered successfully per second"`
	ReplicationLagBytes   float64            `json:"replicationLagBytes"   doc:"Peak replication lag in bytes"`
	MemoryAllocBytes      float64            `json:"memoryAllocBytes"      doc:"Peak allocated heap in bytes"`
	SinkDelivery          map[string]int64   `json:"sinkDelivery"          doc:"Delivered and failed messages by sink type"`
	SinkLatencyP95Seconds map[string]float64 `json:"sinkLatencyP95Seconds" doc:"P95 sink write latency in seconds by sink type; a type without an estimate is absent"`
}

type MetricsResponse struct {
	Body struct {
		Duration      string         `json:"duration"      doc:"Width of the buckets the points are aggregated into"`
		BucketSeconds int            `json:"bucketSeconds" doc:"Bucket width in seconds"`
		Points        []MetricsPoint `json:"points"        doc:"Newest points, oldest first"`
	}
}

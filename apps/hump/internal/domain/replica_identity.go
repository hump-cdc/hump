package domain

// ReplicaIdentity is how hump manages a table's REPLICA IDENTITY, set by the
// table's replica key in YAML.
type ReplicaIdentity string

const (
	// ReplicaIdentityFull has hump set REPLICA IDENTITY FULL on the table, so
	// the WAL carries every column of the old row.
	ReplicaIdentityFull ReplicaIdentity = "full"
	// ReplicaIdentityUnmanaged leaves the table's replica identity alone. Under
	// anything but FULL the WAL omits unchanged TOAST values and non-key columns
	// of the old row; they reach sinks as [UnavailableValue].
	ReplicaIdentityUnmanaged ReplicaIdentity = "unmanaged"
)

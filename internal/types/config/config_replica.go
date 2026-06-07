package config

// ReplicaIdentity controls whether hump manages a table's PostgreSQL replica
// identity. Full makes hump ensure REPLICA IDENTITY FULL on the table and its
// partitions; Unmanaged leaves the table's replica identity untouched.
type ReplicaIdentity string

const (
	ReplicaIdentityFull      ReplicaIdentity = "full"
	ReplicaIdentityUnmanaged ReplicaIdentity = "unmanaged"
)

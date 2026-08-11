package config

type ReplicaIdentity string

const (
	ReplicaIdentityFull      ReplicaIdentity = "full"
	ReplicaIdentityUnmanaged ReplicaIdentity = "unmanaged"
)

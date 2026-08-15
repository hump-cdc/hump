package postgres

import (
	"maps"

	"hump/config"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"
)

// introspectorRegistry maps a table to its configured sinks and resolves
// partition tables back to their parent table configuration.
type introspectorRegistry struct {
	// tables lists every configured table, in configuration order.
	tables []domain.TableIdentifier
	// replicaFull lists the configured tables set to replica: full.
	replicaFull []domain.TableIdentifier
	// partitions maps a partition to its parent; refreshed by UpdatePartitions.
	partitions map[domain.TableIdentifier]domain.TableIdentifier
	// registry maps a configured table to the sinks it feeds.
	registry map[domain.TableIdentifier][]sinkcfg.Config
}

// newIntrospectorRegistry builds a registry from a parsed config.Configuration.
func newIntrospectorRegistry(cfg config.Configuration) *introspectorRegistry {
	result := &introspectorRegistry{
		registry:   make(map[domain.TableIdentifier][]sinkcfg.Config),
		partitions: make(map[domain.TableIdentifier]domain.TableIdentifier),
	}

	for _, t := range cfg.Tables {
		ref := domain.TableIdentifier{Schema: t.Schema, Name: t.Name}

		result.registry[ref] = t.Sinks
		result.tables = append(result.tables, ref)
		if t.Replica == domain.ReplicaIdentityFull {
			result.replicaFull = append(result.replicaFull, ref)
		}
	}

	return result
}

// List returns all configured tables.
func (r *introspectorRegistry) List() []domain.TableIdentifier {
	return r.tables
}

// ReplicaIdentityFullTables returns the configured tables whose replica identity
// hump manages (replica: full). Tables marked replica: unmanaged are excluded, so
// hump issues no REPLICA IDENTITY DDL for them or their partitions.
func (r *introspectorRegistry) ReplicaIdentityFullTables() []domain.TableIdentifier {
	return r.replicaFull
}

// Resolve looks up the sinks for table, falling back to the partition→parent
// mapping when table is not configured directly. The returned TableIdentifier is
// the parent for a partition, table itself for a direct hit, or zero on a miss.
func (r *introspectorRegistry) Resolve(table domain.TableIdentifier) (domain.TableIdentifier, []sinkcfg.Config) {
	if sinks, ok := r.registry[table]; ok {
		return table, sinks
	}
	if parent, ok := r.resolveByPartition(table); ok {
		return r.Resolve(parent)
	}
	return domain.TableIdentifier{}, nil
}

// Partitions returns the partition→parent mapping.
func (r *introspectorRegistry) Partitions() map[domain.TableIdentifier]domain.TableIdentifier {
	return r.partitions
}

// UpdatePartitions replaces the partition→parent mapping. The input is
// copied so the registry does not alias the caller's map.
func (r *introspectorRegistry) UpdatePartitions(partitions map[domain.TableIdentifier]domain.TableIdentifier) {
	result := make(map[domain.TableIdentifier]domain.TableIdentifier, len(partitions))
	maps.Copy(result, partitions)

	r.partitions = result
}

// resolveByPartition returns the parent of a partition, or false when table is
// not a tracked partition.
func (r *introspectorRegistry) resolveByPartition(table domain.TableIdentifier) (domain.TableIdentifier, bool) {
	ref, ok := r.partitions[table]
	return ref, ok
}

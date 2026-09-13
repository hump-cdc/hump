package postgres

import (
	"maps"

	"hump/config"
	"hump/internal/domain"
)

// introspectorRegistry holds the configured tables and resolves partition
// tables back to their parent table configuration.
type introspectorRegistry struct {
	// tables lists every configured table, in configuration order.
	tables []domain.TableIdentifier
	// replicaFull lists the configured tables set to replica: full.
	replicaFull []domain.TableIdentifier
	// partitions maps a partition to its parent; refreshed by UpdatePartitions.
	partitions map[domain.TableIdentifier]domain.TableIdentifier
	// configured is the set of tables named in the configuration.
	configured map[domain.TableIdentifier]struct{}
}

// newIntrospectorRegistry builds a registry from a parsed config.Configuration.
func newIntrospectorRegistry(cfg config.Configuration) *introspectorRegistry {
	result := &introspectorRegistry{
		configured: make(map[domain.TableIdentifier]struct{}),
		partitions: make(map[domain.TableIdentifier]domain.TableIdentifier),
	}

	for _, t := range cfg.Tables {
		ref := domain.TableIdentifier{Schema: t.Schema, Name: t.Name}

		result.configured[ref] = struct{}{}
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

// Resolve returns the configured table that owns table — table itself on a direct
// hit, its parent when table is a tracked partition — or false when neither is
// configured. Resolution is one level deep: only configured tables are recorded as parents.
func (r *introspectorRegistry) Resolve(table domain.TableIdentifier) (domain.TableIdentifier, bool) {
	if _, ok := r.configured[table]; ok {
		return table, true
	}
	if parent, ok := r.resolveByPartition(table); ok {
		if _, ok := r.configured[parent]; ok {
			return parent, true
		}
	}

	return domain.TableIdentifier{}, false
}

// IsConfigured reports whether table is named in the configuration. Unlike
// Resolve it does not consider partitions, so a partition of a configured table
// answers false.
func (r *introspectorRegistry) IsConfigured(table domain.TableIdentifier) bool {
	_, ok := r.configured[table]
	return ok
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

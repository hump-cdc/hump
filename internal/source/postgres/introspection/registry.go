package introspection

import (
	"hump/internal/domain"
	"hump/internal/port/sink"
	"hump/config"
	"hump/internal/types/introspector"
	"maps"
)

// TableRegistry maps schema/table to its configured sinks and resolves
// partition tables back to their parent table configuration.
type TableRegistry struct {
	tables      []introspector.TableRef
	replicaFull []introspector.TableRef
	partitions  map[domain.TableSchema]map[domain.TableName]introspector.TableRef

	// mapping table schema → table name → sinks
	registry map[domain.TableSchema]map[domain.TableName][]sink.Config
}

// NewTableRegistry returns an empty TableRegistry. Use RegistryFromConfig
// when populating from a parsed configuration.
func NewTableRegistry() *TableRegistry {
	return &TableRegistry{
		registry:   make(map[domain.TableSchema]map[domain.TableName][]sink.Config),
		partitions: make(map[domain.TableSchema]map[domain.TableName]introspector.TableRef),
	}
}

// RegistryFromConfig builds a registry from a parsed config.Configuration.
func RegistryFromConfig(cfg config.Configuration) *TableRegistry {
	result := NewTableRegistry()

	for _, t := range cfg.Tables {
		schemaName := domain.TableSchema(t.Schema)
		name := domain.TableName(t.Name)

		tables, ok := result.registry[schemaName]
		if !ok {
			tables = make(map[domain.TableName][]sink.Config)
			result.registry[schemaName] = tables
		}

		tables[name] = t.Sinks
		ref := introspector.TableRef{Schema: schemaName, Name: name}
		result.tables = append(result.tables, ref)
		if t.Replica == domain.ReplicaIdentityFull {
			result.replicaFull = append(result.replicaFull, ref)
		}
	}

	return result
}

// List returns all configured tables (schema and name).
func (r *TableRegistry) List() []introspector.TableRef {
	return r.tables
}

// ReplicaIdentityFullTables returns only the configured tables whose replica
// identity hump manages (replica: full). Tables configured replica: unmanaged
// are excluded so hump issues no REPLICA IDENTITY DDL for them or their
// partitions.
func (r *TableRegistry) ReplicaIdentityFullTables() []introspector.TableRef {
	return r.replicaFull
}

// Resolve looks up sinks for a schema/table. If the table is not configured
// directly, it falls back to the partition→parent mapping. The returned
// TableRef points to the resolved parent table when the input was a partition,
// or to the input table itself when it was configured directly.
func (r *TableRegistry) Resolve(schemaName domain.TableSchema, name domain.TableName) (introspector.TableRef, []sink.Config) {
	if s, ok := r.registry[schemaName]; ok {
		if sinks, ok := s[name]; ok {
			return introspector.TableRef{Schema: schemaName, Name: name}, sinks
		}
	}
	if parent, ok := r.resolveByPartition(schemaName, name); ok {
		return r.Resolve(parent.Schema, parent.Name)
	}
	return introspector.TableRef{}, nil
}

// Partitions returns the partition→parent mapping.
func (r *TableRegistry) Partitions() map[domain.TableSchema]map[domain.TableName]introspector.TableRef {
	return r.partitions
}

// UpdatePartitions replaces the partition→parent mapping. The input is
// copied so the registry does not alias the caller's map.
func (r *TableRegistry) UpdatePartitions(partitions map[domain.TableSchema]map[domain.TableName]introspector.TableRef) {
	result := make(map[domain.TableSchema]map[domain.TableName]introspector.TableRef, len(partitions))

	for schemaName, tables := range partitions {
		c := make(map[domain.TableName]introspector.TableRef, len(tables))
		maps.Copy(c, tables)
		result[schemaName] = c
	}

	r.partitions = result
}

// resolveByPartition returns the parent TableRef for a partition, or false
// when schemaName/name is not a tracked partition.
func (r *TableRegistry) resolveByPartition(schemaName domain.TableSchema, name domain.TableName) (introspector.TableRef, bool) {
	if s, ok := r.partitions[schemaName]; ok {
		if ref, ok := s[name]; ok {
			return ref, true
		}
	}
	return introspector.TableRef{}, false
}

package introspection

import (
	"hump/internal/types/config"
	"hump/internal/types/introspector"
	"maps"
)

// TableRegistry maps schema/table to its configured sinks and resolves
// partition tables back to their parent table configuration.
type TableRegistry struct {
	tables     []introspector.TableRef
	partitions map[introspector.TableSchema]map[introspector.TableName]introspector.TableRef

	// mapping table schema → table name → sinks
	registry map[introspector.TableSchema]map[introspector.TableName][]config.SinkConfig
}

// NewTableRegistry returns an empty TableRegistry. Use RegistryFromConfig
// when populating from a parsed configuration.
func NewTableRegistry() *TableRegistry {
	return &TableRegistry{
		registry:   make(map[introspector.TableSchema]map[introspector.TableName][]config.SinkConfig),
		partitions: make(map[introspector.TableSchema]map[introspector.TableName]introspector.TableRef),
	}
}

// RegistryFromConfig builds a registry from a parsed config.Configuration.
func RegistryFromConfig(cfg config.Configuration) *TableRegistry {
	result := NewTableRegistry()

	for _, t := range cfg.Tables {
		tables, ok := result.registry[t.Schema]
		if !ok {
			tables = make(map[introspector.TableName][]config.SinkConfig)
			result.registry[t.Schema] = tables
		}

		tables[t.Name] = t.Sinks
		result.tables = append(result.tables, introspector.TableRef{Schema: t.Schema, Name: t.Name})
	}

	return result
}

// List returns all configured tables (schema and name).
func (r *TableRegistry) List() []introspector.TableRef {
	return r.tables
}

// Resolve looks up sinks for a schema/table. If the table is not configured
// directly, it falls back to the partition→parent mapping. The returned
// TableRef points to the resolved parent table when the input was a partition,
// or to the input table itself when it was configured directly.
func (r *TableRegistry) Resolve(schemaName, name string) (introspector.TableRef, []config.SinkConfig) {
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
func (r *TableRegistry) Partitions() map[introspector.TableSchema]map[introspector.TableName]introspector.TableRef {
	return r.partitions
}

// UpdatePartitions replaces the partition→parent mapping. The input is
// copied so the registry does not alias the caller's map.
func (r *TableRegistry) UpdatePartitions(partitions map[introspector.TableSchema]map[introspector.TableName]introspector.TableRef) {
	result := make(map[introspector.TableSchema]map[introspector.TableName]introspector.TableRef, len(partitions))

	for schemaName, tables := range partitions {
		c := make(map[introspector.TableName]introspector.TableRef, len(tables))
		maps.Copy(c, tables)
		result[schemaName] = c
	}

	r.partitions = result
}

// resolveByPartition returns the parent TableRef for a partition, or false
// when schemaName/name is not a tracked partition.
func (r *TableRegistry) resolveByPartition(schemaName, name string) (introspector.TableRef, bool) {
	if s, ok := r.partitions[schemaName]; ok {
		if ref, ok := s[name]; ok {
			return ref, true
		}
	}
	return introspector.TableRef{}, false
}

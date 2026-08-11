package config

import (
	"hump/internal/domain"
	"hump/internal/port/sink"
	pkgconfig "hump/pkg/config"

	"github.com/goccy/go-yaml/ast"
)

type Configuration struct {
	registry map[domain.TableSchema]map[domain.TableName][]sink.Config `yaml:"-"`
	Tables   []Table                                                   `yaml:"tables" validate:"min=1"`
}

func (c *Configuration) buildRegistry() {
	c.registry = make(map[domain.TableSchema]map[domain.TableName][]sink.Config)

	for _, table := range c.Tables {
		tables, ok := c.registry[table.Schema]
		if !ok {
			tables = make(map[domain.TableName][]sink.Config)
			c.registry[table.Schema] = tables
		}

		tables[table.Name] = table.Sinks
	}
}

func (s *Configuration) Resolve(schema domain.TableSchema, table domain.TableName) []sink.Config {
	if s, ok := s.registry[schema]; ok {
		if sinks, ok := s[table]; ok {
			return sinks
		}
	}

	return nil
}

type Table struct {
	Schema  domain.TableSchema     `yaml:"schema"            validate:"omitempty"`
	Name    domain.TableName       `yaml:"table"             validate:"required"`
	Replica domain.ReplicaIdentity `yaml:"replica,omitempty" validate:"omitempty,oneof=full unmanaged"`
	Sinks   []sink.Config          `yaml:"sinks"             validate:"min=1"`
}

func (t *Table) UnmarshalYAML(node ast.Node) error {
	type table Table
	if err := pkgconfig.Decode(node, (*table)(t)); err != nil {
		return err
	}

	t.setDefaults()

	return nil
}

func (t *Table) setDefaults() {
	if t.Replica == "" {
		t.Replica = domain.ReplicaIdentityFull
	}
}

func crossValidate(tables []Table) error {
	groups := make([][]sink.Config, 0, len(tables))
	for _, table := range tables {
		groups = append(groups, table.Sinks)
	}

	return sink.CrossValidate(groups)
}

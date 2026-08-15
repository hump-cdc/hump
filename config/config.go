package config

import (
	"hump/config/decoder"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"

	"github.com/goccy/go-yaml/ast"
)

type Configuration struct {
	registry map[domain.TableIdentifier][]sinkcfg.Config `yaml:"-"`
	Tables   []Table                                     `yaml:"tables" validate:"min=1"`
}

func (c *Configuration) buildRegistry() {
	c.registry = make(map[domain.TableIdentifier][]sinkcfg.Config, len(c.Tables))

	for _, table := range c.Tables {
		c.registry[domain.TableIdentifier{Schema: table.Schema, Name: table.Name}] = table.Sinks
	}
}

func (c *Configuration) ResolveAll() []Table {
	return c.Tables
}

func (c *Configuration) Resolve(id domain.TableIdentifier) []sinkcfg.Config {
	return c.registry[id]
}

type Table struct {
	Schema  domain.TableSchema     `yaml:"schema"            validate:"omitempty"`
	Name    domain.TableName       `yaml:"table"             validate:"required"`
	Replica domain.ReplicaIdentity `yaml:"replica,omitempty" validate:"omitempty,oneof=full unmanaged"`
	Sinks   []sinkcfg.Config       `yaml:"sinks"             validate:"min=1"`
}

func (t *Table) UnmarshalYAML(node ast.Node) error {
	type table Table
	if err := decoder.Decode(node, (*table)(t)); err != nil {
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
	groups := make([][]sinkcfg.Config, 0, len(tables))
	for _, table := range tables {
		groups = append(groups, table.Sinks)
	}

	return sinkcfg.CrossValidate(groups)
}

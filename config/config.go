// Package config turns the YAML table configuration into a validated
// [Configuration] with every default filled in; [Parse] is its entry point.
package config

import (
	"hump/config/decoder"
	sinkcfg "hump/config/sink"
	"hump/internal/domain"

	"github.com/goccy/go-yaml/ast"
)

// Configuration is the parsed configuration: the tables hump replicates.
type Configuration struct {
	Tables []Table `yaml:"tables" validate:"min=1"`
}

// ResolveAll returns every configured table.
func (c *Configuration) ResolveAll() []Table {
	return c.Tables
}

// Table is one replicated table and the sinks its changes are delivered to.
type Table struct {
	Schema domain.TableSchema `yaml:"schema"            validate:"omitempty"`
	Name   domain.TableName   `yaml:"table"             validate:"required"`
	// Replica is the replica identity hump enforces; "full" (the default) is
	// what makes unchanged TOAST values and old rows arrive complete.
	Replica domain.ReplicaIdentity `yaml:"replica,omitempty" validate:"omitempty,oneof=full unmanaged"`
	Sinks   []sinkcfg.Config       `yaml:"sinks"             validate:"min=1"`
}

// UnmarshalYAML decodes a table strictly and fills in its defaults.
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

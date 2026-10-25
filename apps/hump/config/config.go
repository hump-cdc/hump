// Package config turns the YAML table configuration into a validated
// [Configuration] with every default filled in; [Parse] is its entry point.
package config

import (
	"fmt"

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

	if err := t.checkSinks(node); err != nil {
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

// checkSinks rejects an empty sink entry such as `sinks: [~]`. The decoder
// leaves it a zero sinkcfg.Config without calling its UnmarshalYAML, so it
// would otherwise reach the bindings with a nil Sink, Failure and Retry.
func (t *Table) checkSinks(node ast.Node) error {
	var items []ast.Node
	if mapping, ok := node.(*ast.MappingNode); ok {
		if sinks, ok := decoder.KeyValue(mapping, "sinks").(*ast.SequenceNode); ok {
			items = sinks.Values
		}
	}

	for i, sink := range t.Sinks {
		if sink.Sink != nil {
			continue
		}

		line := decoder.NodeLine(node)
		if i < len(items) {
			line = decoder.NodeLine(items[i])
		}

		return fmt.Errorf("line %d: sink entry %d of table %s is empty", line, i+1, t.displayName())
	}

	return nil
}

// displayName returns the table as schema.table, or just table without a schema.
func (t *Table) displayName() string {
	if t.Schema == "" {
		return string(t.Name)
	}

	return string(t.Schema) + "." + string(t.Name)
}

func crossValidate(tables []Table) error {
	groups := make([][]sinkcfg.Config, 0, len(tables))
	for _, table := range tables {
		groups = append(groups, table.Sinks)
	}

	return sinkcfg.CrossValidate(groups)
}

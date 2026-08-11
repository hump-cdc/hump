package config

import (
	"hump/internal/port/sink"

	"github.com/goccy/go-yaml/ast"
)

type Configuration struct {
	Tables []Tables `yaml:"tables" validate:"min=1"`
}

type Tables struct {
	Schema  string          `yaml:"schema"            validate:"omitempty"`
	Name    string          `yaml:"table"             validate:"required"`
	Replica ReplicaIdentity `yaml:"replica,omitempty" validate:"omitempty,oneof=full unmanaged"`
	Sinks   []sink.Config   `yaml:"sinks"             validate:"min=1"`
}

func (t *Tables) UnmarshalYAML(node ast.Node) error {
	type tables Tables
	if err := decode(node, (*tables)(t)); err != nil {
		return err
	}

	t.setDefaults()

	return nil
}

func (t *Tables) setDefaults() {
	if t.Replica == "" {
		t.Replica = ReplicaIdentityFull
	}
}

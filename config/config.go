package config

import (
	"hump/internal/domain"
	"hump/internal/port/sink"
	pkgconfig "hump/pkg/config"

	"github.com/goccy/go-yaml/ast"
)

type Configuration struct {
	Tables []Tables `yaml:"tables" validate:"min=1"`
}

type Tables struct {
	Schema  string                 `yaml:"schema"            validate:"omitempty"`
	Name    string                 `yaml:"table"             validate:"required"`
	Replica domain.ReplicaIdentity `yaml:"replica,omitempty" validate:"omitempty,oneof=full unmanaged"`
	Sinks   []sink.Config          `yaml:"sinks"             validate:"min=1"`
}

func (t *Tables) UnmarshalYAML(node ast.Node) error {
	type tables Tables
	if err := pkgconfig.Decode(node, (*tables)(t)); err != nil {
		return err
	}

	t.setDefaults()

	return nil
}

func (t *Tables) setDefaults() {
	if t.Replica == "" {
		t.Replica = domain.ReplicaIdentityFull
	}
}

func CrossValidate(tables []Tables) error {
	groups := make([][]sink.Config, 0, len(tables))
	for _, table := range tables {
		groups = append(groups, table.Sinks)
	}

	return sink.CrossValidate(groups)
}

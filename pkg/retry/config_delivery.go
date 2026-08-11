package retry

import (
	_ "embed"

	"hump/pkg/config"
	"hump/pkg/template"

	"github.com/goccy/go-yaml/ast"
)

var (
	//go:embed templ/concurrency_key.tmpl
	defaultConcurrencyKey string
)

type DeliveryMode string

const (
	DeliveryModeReliable DeliveryMode = "reliable"
	DeliveryModeAsync    DeliveryMode = "async"
)

type Delivery struct {
	Mode           DeliveryMode       `yaml:"mode,omitempty"           validate:"omitempty,oneof=reliable async"`
	MaxConcurrent  int                `yaml:"maxConcurrent,omitempty"  validate:"gte=0"`
	ConcurrencyKey *template.Template `yaml:"concurrencyKey,omitempty"`
}

func (d *Delivery) UnmarshalYAML(node ast.Node) error {
	type delivery Delivery
	if err := config.Decode(node, (*delivery)(d)); err != nil {
		return err
	}

	d.SetDefaults()

	return nil
}

func (d *Delivery) SetDefaults() {
	if d.Mode == "" {
		d.Mode = DeliveryModeReliable
	}

	if d.MaxConcurrent <= 0 {
		d.MaxConcurrent = 64
	}

	if d.ConcurrencyKey == nil {
		concurrentKey, _ := template.New(defaultConcurrencyKey)
		d.ConcurrencyKey = concurrentKey
	}
}

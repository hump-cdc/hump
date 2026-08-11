package config

import (
	pkgconfig "hump/pkg/config"

	"github.com/goccy/go-yaml"
)

// Parse parses raw YAML into a Configuration. The returned error, when
// non-nil, is a goccy yaml.Error whose Error()/FormatError output already
// includes line:column and a source snippet, or an error wrapping
// [errs.ErrCrossValidate] describing inter-sink configuration divergences.
func Parse(source string) (Configuration, error) {
	expaned, err := expand(source)
	if err != nil {
		return Configuration{}, err
	}

	var cfg Configuration
	if err := yaml.UnmarshalWithOptions(
		[]byte(expaned), &cfg,
		yaml.Strict(),
		yaml.Validator(pkgconfig.Validate),
	); err != nil {
		return Configuration{}, err
	}

	if err := CrossValidate(cfg.Tables); err != nil {
		return Configuration{}, err
	}

	return cfg, nil
}

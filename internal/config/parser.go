package config

import (
	"hump/internal/types/config"

	"github.com/goccy/go-yaml"
)

// Parse parses raw YAML into a Configuration. The returned error, when
// non-nil, is a goccy yaml.Error whose Error()/FormatError output already
// includes line:column and a source snippet, or an error wrapping
// [errs.ErrCrossValidate] describing inter-sink configuration divergences.
func Parse(source string) (config.Configuration, error) {
	expaned, err := expand(source)
	if err != nil {
		return config.Configuration{}, err
	}

	var cfg config.Configuration
	if err := yaml.UnmarshalWithOptions(
		[]byte(expaned), &cfg,
		yaml.Strict(),
		yaml.Validator(config.Validate),
	); err != nil {
		return config.Configuration{}, err
	}

	if err := config.CrossValidate(cfg.Tables); err != nil {
		return config.Configuration{}, err
	}

	return cfg, nil
}

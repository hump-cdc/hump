package config

import (
	"hump/config/decoder"

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
		yaml.Validator(decoder.Validate),
	); err != nil {
		return Configuration{}, err
	}

	if err := crossValidate(cfg.Tables); err != nil {
		return Configuration{}, err
	}

	cfg.buildRegistry()

	return cfg, nil
}

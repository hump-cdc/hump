package config

import (
	"hump/config/decoder"
)

// Parse parses raw YAML into a Configuration. The returned error, when
// non-nil, is a goccy yaml.Error whose Error()/FormatError output already
// includes line:column and a source snippet, a validator error without a
// position for a document with no tables at all, or an error wrapping
// [hump/config/errors.ErrCrossValidate] describing inter-sink configuration
// divergences.
func Parse(source string) (Configuration, error) {
	expaned, err := expand(source)
	if err != nil {
		return Configuration{}, err
	}

	var cfg Configuration
	if err := decoder.Unmarshal([]byte(expaned), &cfg); err != nil {
		return Configuration{}, err
	}

	// The decoder validates only a mapping, so an empty or comment-only
	// document reaches here unvalidated and with no tables.
	if len(cfg.Tables) == 0 {
		if err := decoder.Validate.Struct(&cfg); err != nil {
			return Configuration{}, err
		}
	}

	if err := crossValidate(cfg.Tables); err != nil {
		return Configuration{}, err
	}

	return cfg, nil
}

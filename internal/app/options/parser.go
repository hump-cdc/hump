package options

import (
	"errors"
	"io"
	"os"

	"github.com/jessevdk/go-flags"
)

// resolveConfig populates Config from --config, --config-file, the CONFIG and
// CONFIG_FILE environment variables, or stdin (in that order). It returns
// ErrConfigConflict if both --config and --config-file are set, and
// ErrConfigMissing if no source yields a value.
func (o *Options) resolveConfig() error {
	// error if both config and config path are provided
	if o.Config != "" && o.ConfigPath != "" {
		return ErrConfigConflict
	}

	// resolve from env if not provided via flags
	if o.Config == "" && o.ConfigPath == "" {
		o.Config = os.Getenv("CONFIG")

		if o.Config == "" {
			o.ConfigPath = os.Getenv("CONFIG_FILE")
		}
	}

	// resolve from file if config path is provided
	if o.ConfigPath != "" {
		content, err := os.ReadFile(o.ConfigPath)
		if err != nil {
			return err
		}

		o.Config = string(content)
	}

	// resolve from stdin if config is not provided via flags or env
	if o.Config == "" && o.ConfigPath == "" {
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			content, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}

			o.Config = string(content)
		}
	}

	// error if config is not provided via any method
	if o.Config == "" && o.ConfigPath == "" {
		return ErrConfigMissing
	}

	return nil
}

// Parse reads command-line flags and environment variables into Options and
// resolves the configuration source. It returns ErrHelp when -h/--help is
// requested, ErrConfigConflict, ErrConfigMissing, or any underlying I/O error.
func Parse() (Options, error) {
	var o Options

	parser := flags.NewParser(&o, flags.HelpFlag|flags.PassDoubleDash)

	_, err := parser.Parse()
	if err != nil {
		var fErr *flags.Error
		if errors.As(err, &fErr) && fErr.Type == flags.ErrHelp {
			return o, ErrHelp
		}

		return o, err
	}

	if err = o.resolveConfig(); err != nil {
		return o, err
	}

	return o, nil
}

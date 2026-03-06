package config

type Configuration struct {
	Tables []Tables `yaml:"tables" validate:"min=1"`
}

type Tables struct {
	Schema string       `yaml:"schema"            validate:"omitempty"`
	Name   string       `yaml:"table"             validate:"required"`
	Sinks  []SinkConfig `yaml:"sinks"             validate:"min=1"`
}

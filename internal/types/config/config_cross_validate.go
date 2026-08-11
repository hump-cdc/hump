package config

import "hump/internal/port/sink"

func CrossValidate(tables []Tables) error {
	groups := make([][]sink.Config, 0, len(tables))
	for _, table := range tables {
		groups = append(groups, table.Sinks)
	}

	return sink.CrossValidate(groups)
}

package connection

import (
	"fmt"
	"net/url"
)

// withReplication returns source with the replication=database query
// parameter set, enabling logical replication on the resulting session.
// It rejects schemes other than postgres and postgresql.
func withReplication(source string) (string, error) {
	connectionString, err := url.Parse(source)
	if err != nil {
		return "", err
	}

	if connectionString.Scheme != "postgres" && connectionString.Scheme != "postgresql" {
		return "", fmt.Errorf("unsupported connection string scheme %q", connectionString.Scheme)
	}

	query := connectionString.Query()
	query.Set("replication", "database")
	connectionString.RawQuery = query.Encode()

	return connectionString.String(), nil
}

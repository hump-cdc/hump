package postgres

// GetDbName returns the database name parsed from the connection string, or an
// empty string when the string is unusable.
func (p *Postgres) GetDbName() string {
	return p.dbName
}

// GetDbHost returns the host parsed from the connection string, or an empty
// string when the string is unusable.
func (p *Postgres) GetDbHost() string {
	return p.dbHost
}

// GetDbPort returns the port parsed from the connection string, or zero when
// the string is unusable.
func (p *Postgres) GetDbPort() uint16 {
	return p.dbPort
}

// GetPublicationName returns the publication this adapter replicates through.
func (p *Postgres) GetPublicationName() string {
	return p.publicationName
}

// GetSlotName returns the replication slot this adapter streams from.
func (p *Postgres) GetSlotName() string {
	return p.slotName
}

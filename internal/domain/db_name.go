// Package domain is the vocabulary every layer shares: the named types and
// constants a change event is made of, with no behaviour and no imports.
package domain

// DbName is the name of the database a change event came from.
type DbName string

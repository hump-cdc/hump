package postgres

import (
	"strings"
)

// snapshotWhereRecentInterval is how far back a suggested filter reaches.
const snapshotWhereRecentInterval = "7 days"

const (
	// snapshotWhereRankTimeKey ranks the time dimension or range partitioning
	// key, which also lets the server prune chunks and partitions.
	snapshotWhereRankTimeKey = iota
	// snapshotWhereRankUpdated ranks columns named after the last change of a row.
	snapshotWhereRankUpdated
	// snapshotWhereRankCreated ranks columns named after the creation of a row.
	snapshotWhereRankCreated
	// snapshotWhereRankEvent ranks columns named after a generic event time.
	snapshotWhereRankEvent
	// snapshotWhereRankNowDefault ranks the only temporal column defaulting to
	// the current time.
	snapshotWhereRankNowDefault
)

// snapshotWhereNameRanks ranks temporal columns by their normalized name.
var snapshotWhereNameRanks = map[string]int{
	"updatedat": snapshotWhereRankUpdated, "updated": snapshotWhereRankUpdated,
	"updatedon": snapshotWhereRankUpdated, "updatetime": snapshotWhereRankUpdated,
	"updatedate": snapshotWhereRankUpdated, "updatedtime": snapshotWhereRankUpdated,
	"modifiedat": snapshotWhereRankUpdated, "modified": snapshotWhereRankUpdated,
	"modifiedon": snapshotWhereRankUpdated, "lastmodified": snapshotWhereRankUpdated,
	"lastmodifiedat": snapshotWhereRankUpdated, "lastupdated": snapshotWhereRankUpdated,
	"lastupdatedat": snapshotWhereRankUpdated, "changedat": snapshotWhereRankUpdated,
	"changed": snapshotWhereRankUpdated, "mtime": snapshotWhereRankUpdated,

	"createdat": snapshotWhereRankCreated, "created": snapshotWhereRankCreated,
	"createdon": snapshotWhereRankCreated, "createtime": snapshotWhereRankCreated,
	"createdate": snapshotWhereRankCreated, "createdtime": snapshotWhereRankCreated,
	"creationdate": snapshotWhereRankCreated, "creationtime": snapshotWhereRankCreated,
	"insertedat": snapshotWhereRankCreated, "inserted": snapshotWhereRankCreated,
	"addedat": snapshotWhereRankCreated, "ctime": snapshotWhereRankCreated,

	"timestamp": snapshotWhereRankEvent, "ts": snapshotWhereRankEvent,
	"time": snapshotWhereRankEvent, "eventtime": snapshotWhereRankEvent,
	"eventat": snapshotWhereRankEvent, "eventtimestamp": snapshotWhereRankEvent,
	"occurredat": snapshotWhereRankEvent, "recordedat": snapshotWhereRankEvent,
	"loggedat": snapshotWhereRankEvent, "datetime": snapshotWhereRankEvent,
	"date": snapshotWhereRankEvent,
}

// snapshotWhereSuggester suggests a snapshot filter from the columns of one table.
type snapshotWhereSuggester struct {
	columns        []tablesColumn
	soleNowDefault bool // soleNowDefault is set when exactly one temporal column defaults to the current time.
}

// newSnapshotWhereSuggester returns a suggester over columns.
func newSnapshotWhereSuggester(columns []tablesColumn) *snapshotWhereSuggester {
	s := &snapshotWhereSuggester{columns: columns}

	nowDefaults := 0
	for _, c := range columns {
		if s.isTemporal(c.Type) && c.NowDefault {
			nowDefaults++
		}
	}
	s.soleNowDefault = nowDefaults == 1

	return s
}

// Suggest returns a filter keeping the rows of the last snapshotWhereRecentInterval by
// the temporal column most likely to track row recency, or an empty string
// when no column qualifies. Ties go to an indexed column, then the earlier one.
func (s *snapshotWhereSuggester) Suggest() string {
	var best *tablesColumn
	bestRank := 0
	for i := range s.columns {
		c := &s.columns[i]

		rank, ok := s.rank(c)
		if !ok {
			continue
		}

		if best == nil || rank < bestRank || (rank == bestRank && c.Indexed && !best.Indexed) {
			best, bestRank = c, rank
		}
	}

	if best == nil {
		return ""
	}

	return quoteIdentifier(best.Name) + " >= now() - interval " + quoteLiteral(snapshotWhereRecentInterval)
}

// rank returns how well c tracks row recency, lower being better, and false
// when c is not a candidate. A column is admitted on its default alone only
// when no other column shares it.
func (s *snapshotWhereSuggester) rank(c *tablesColumn) (int, bool) {
	if !s.isTemporal(c.Type) {
		return 0, false
	}

	if c.TimeKey {
		return snapshotWhereRankTimeKey, true
	}

	if rank, ok := snapshotWhereNameRanks[s.normalizeColumnName(c.Name)]; ok {
		return rank, true
	}

	if c.NowDefault && s.soleNowDefault {
		return snapshotWhereRankNowDefault, true
	}

	return 0, false
}

// isTemporal reports whether a type, as format_type renders it, is a date or a
// timestamp with or without time zone and precision.
func (s *snapshotWhereSuggester) isTemporal(typ string) bool {
	return typ == "date" || strings.HasPrefix(typ, "timestamp")
}

// normalizeColumnName lowercases name and drops the separators naming
// conventions differ by, so updated_at, updatedAt and "Updated At" compare equal.
func (s *snapshotWhereSuggester) normalizeColumnName(name string) string {
	return strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(name))
}

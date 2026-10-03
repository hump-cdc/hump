package postgres

import (
	"strings"
)

// whereRecentInterval is how far back a suggested filter reaches.
const whereRecentInterval = "7 days"

const (
	// whereRankTimeKey ranks the time dimension or range partitioning key, which
	// also lets the server prune chunks and partitions.
	whereRankTimeKey = iota
	// whereRankUpdated ranks columns named after the last change of a row.
	whereRankUpdated
	// whereRankCreated ranks columns named after the creation of a row.
	whereRankCreated
	// whereRankEvent ranks columns named after a generic event time.
	whereRankEvent
	// whereRankNowDefault ranks the only temporal column defaulting to the current time.
	whereRankNowDefault
)

// whereNameRanks ranks temporal columns by their normalized name.
var whereNameRanks = map[string]int{
	"updatedat": whereRankUpdated, "updated": whereRankUpdated, "updatedon": whereRankUpdated,
	"updatetime": whereRankUpdated, "updatedate": whereRankUpdated, "updatedtime": whereRankUpdated,
	"modifiedat": whereRankUpdated, "modified": whereRankUpdated, "modifiedon": whereRankUpdated,
	"lastmodified": whereRankUpdated, "lastmodifiedat": whereRankUpdated, "lastupdated": whereRankUpdated,
	"lastupdatedat": whereRankUpdated, "changedat": whereRankUpdated, "changed": whereRankUpdated,
	"mtime": whereRankUpdated,

	"createdat": whereRankCreated, "created": whereRankCreated, "createdon": whereRankCreated,
	"createtime": whereRankCreated, "createdate": whereRankCreated, "createdtime": whereRankCreated,
	"creationdate": whereRankCreated, "creationtime": whereRankCreated, "insertedat": whereRankCreated,
	"inserted": whereRankCreated, "addedat": whereRankCreated, "ctime": whereRankCreated,

	"timestamp": whereRankEvent, "ts": whereRankEvent, "time": whereRankEvent,
	"eventtime": whereRankEvent, "eventat": whereRankEvent, "eventtimestamp": whereRankEvent,
	"occurredat": whereRankEvent, "recordedat": whereRankEvent, "loggedat": whereRankEvent,
	"datetime": whereRankEvent, "date": whereRankEvent,
}

// suggestWhere returns a filter keeping the rows of the last whereRecentInterval
// by the temporal column most likely to track row recency, or an empty string
// when no column qualifies. Ties go to an indexed column, then the earlier one.
func suggestWhere(columns []tablesColumn) string {
	nowDefaults := 0
	for _, c := range columns {
		if isTemporal(c.Type) && c.NowDefault {
			nowDefaults++
		}
	}

	var best *tablesColumn
	bestRank := 0
	for i := range columns {
		c := &columns[i]

		rank, ok := whereRank(c, nowDefaults == 1)
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

	return quoteIdentifier(best.Name) + " >= now() - interval " + quoteLiteral(whereRecentInterval)
}

// whereRank returns how well c tracks row recency, lower being better, and
// false when c is not a candidate. soleNowDefault admits a column on its
// default alone, which is only telling when no other column shares it.
func whereRank(c *tablesColumn, soleNowDefault bool) (int, bool) {
	if !isTemporal(c.Type) {
		return 0, false
	}

	if c.TimeKey {
		return whereRankTimeKey, true
	}

	if rank, ok := whereNameRanks[normalizeColumnName(c.Name)]; ok {
		return rank, true
	}

	if c.NowDefault && soleNowDefault {
		return whereRankNowDefault, true
	}

	return 0, false
}

// isTemporal reports whether a type, as format_type renders it, is a date or a
// timestamp with or without time zone and precision.
func isTemporal(typ string) bool {
	return typ == "date" || strings.HasPrefix(typ, "timestamp")
}

// normalizeColumnName lowercases name and drops the separators naming
// conventions differ by, so updated_at, updatedAt and "Updated At" compare equal.
func normalizeColumnName(name string) string {
	return strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(name))
}

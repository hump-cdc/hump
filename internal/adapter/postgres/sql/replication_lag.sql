SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), confirmed_flush_lsn)
FROM pg_replication_slots
WHERE slot_type = 'logical' AND slot_name = $1

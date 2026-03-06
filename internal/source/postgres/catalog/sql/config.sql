SELECT
    current_setting('server_version_num'),
    current_setting('wal_level'),
    current_setting('max_replication_slots'),
    current_setting('max_wal_senders'),
    EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')

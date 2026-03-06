SELECT name, setting AS value
FROM pg_settings
WHERE name IN ('wal_level', 'max_replication_slots', 'max_wal_senders', 'timescaledb.enable_hypertable_create');

SELECT slot_name, active, restart_lsn,
       pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) AS lag
FROM pg_replication_slots;

WITH pub_info AS (
    SELECT
        p.oid,
        p.pubname,
        r.rolname AS owner,
        p.puballtables,
        p.pubinsert,
        p.pubupdate,
        p.pubdelete,
        p.pubtruncate
    FROM pg_publication p
    JOIN pg_roles r ON p.pubowner = r.oid
),
pub_tables AS (
    SELECT
        pubname,
        COUNT(*) AS table_count,
        string_agg(schemaname || '.' || tablename, ', ' ORDER BY schemaname, tablename) AS tables
    FROM pg_publication_tables
    GROUP BY pubname
),
slot_usage AS (
    SELECT
        slot_name,
        active,
        pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(), restart_lsn)) AS lag
    FROM pg_replication_slots
    WHERE slot_type = 'logical'
)
SELECT
    pi.pubname,
    pi.owner,
    COALESCE(pt.tables, '') AS tables,
    (SELECT COUNT(*) FROM slot_usage) AS total_slots
FROM pub_info pi
LEFT JOIN pub_tables pt ON pi.pubname = pt.pubname
ORDER BY pi.pubname;

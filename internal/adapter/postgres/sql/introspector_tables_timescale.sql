-- regular tables
SELECT
    c.relnamespace::regnamespace::text AS schema,
    c.relname AS name,
    'regular' AS type,
    NULL AS partition_schema,
    NULL AS partition_table
FROM pg_class c
LEFT JOIN _timescaledb_catalog.hypertable ht
    ON ht.table_name = c.relname
    AND ht.schema_name = c.relnamespace::regnamespace::text
WHERE (c.relnamespace::regnamespace::text, c.relname::text) IN (
        SELECT s, n FROM unnest($1::text[], $2::text[]) AS f(s, n)
    )
    AND c.relkind = 'r'
    AND ht.id IS NULL
    AND NOT EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid = c.oid)

UNION ALL

-- hypertable
SELECT
    c.relnamespace::regnamespace::text AS schema,
    c.relname AS name,
    'hypertable' AS type,
    COALESCE(ch.schema_name, ht.associated_schema_name) AS partition_schema,
    ch.table_name AS partition_table
FROM pg_class c
LEFT JOIN _timescaledb_catalog.hypertable ht
    ON ht.table_name = c.relname
    AND ht.schema_name = c.relnamespace::regnamespace::text
LEFT JOIN _timescaledb_catalog.chunk ch ON ch.hypertable_id = ht.id
WHERE (ht.schema_name::text, ht.table_name::text) IN (
        SELECT s, n FROM unnest($1::text[], $2::text[]) AS f(s, n)
    )

UNION ALL

-- partitioned
SELECT
    parent.relnamespace::regnamespace::text AS schema,
    parent.relname::text AS name,
    'partitioned' AS type,
    child.relnamespace::regnamespace::text AS partition_schema,
    child.relname AS partition_table
FROM pg_inherits
JOIN pg_class child ON pg_inherits.inhrelid = child.oid
JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
WHERE parent.relkind = 'p'
    AND (parent.relnamespace::regnamespace::text, parent.relname::text) IN (
        SELECT s, n FROM unnest($1::text[], $2::text[]) AS f(s, n)
    )

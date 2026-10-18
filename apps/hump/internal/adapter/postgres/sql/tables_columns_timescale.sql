-- tables: columns
SELECT
    n.nspname AS schema,
    c.relname AS name,
    CASE
        WHEN ht.id IS NOT NULL THEN 'hypertable'
        WHEN c.relkind = 'p' THEN 'partitioned'
        ELSE 'regular'
    END AS type,
    a.attname AS column_name,
    format_type(a.atttypid, a.atttypmod) AS column_type,
    NOT a.attnotnull AS nullable,
    EXISTS (
        SELECT 1 FROM pg_index i
        WHERE i.indrelid = c.oid AND i.indisprimary AND a.attnum = ANY (i.indkey)
    ) AS primary_key,
    EXISTS (
        SELECT 1 FROM _timescaledb_catalog.dimension dim
        WHERE dim.hypertable_id = ht.id AND dim.interval_length IS NOT NULL AND dim.column_name = a.attname
    ) OR EXISTS (
        SELECT 1 FROM pg_partitioned_table pt
        WHERE pt.partrelid = c.oid AND pt.partstrat = 'r' AND pt.partattrs[0] = a.attnum
    ) AS time_key,
    COALESCE(
        pg_get_expr(d.adbin, d.adrelid) ~* '(now\(\)|current_timestamp|localtimestamp|statement_timestamp|transaction_timestamp|clock_timestamp)',
        false
    ) AS now_default,
    EXISTS (
        SELECT 1 FROM pg_index i
        WHERE i.indrelid = c.oid AND i.indkey[0] = a.attnum
    ) AS indexed
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
LEFT JOIN pg_attrdef d ON d.adrelid = c.oid AND d.adnum = a.attnum
LEFT JOIN _timescaledb_catalog.hypertable ht
    ON ht.schema_name = n.nspname AND ht.table_name = c.relname
WHERE (n.nspname::text, c.relname::text) IN (
        SELECT s, n FROM unnest($1::text[], $2::text[]) AS f(s, n)
    )
    AND (
        c.relkind = 'p'
        OR (c.relkind = 'r' AND NOT EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid = c.oid))
    )
ORDER BY n.nspname, c.relname, a.attnum

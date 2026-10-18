-- published tables whose replica identity cannot carry an UPDATE or a DELETE
SELECT
    pt.schemaname::text AS schema,
    pt.tablename::text AS name
FROM pg_publication_tables pt
JOIN pg_namespace n ON n.nspname = pt.schemaname
JOIN pg_class c ON c.relname = pt.tablename AND c.relnamespace = n.oid
WHERE pt.pubname = $1
    AND c.relkind IN ('r', 'p')
    AND CASE c.relreplident
        WHEN 'n' THEN true
        WHEN 'd' THEN NOT EXISTS (
            SELECT 1 FROM pg_index x
            WHERE x.indrelid = c.oid AND x.indisprimary AND x.indisvalid
        )
        WHEN 'i' THEN NOT EXISTS (
            SELECT 1 FROM pg_index x
            WHERE x.indrelid = c.oid AND x.indisreplident AND x.indisvalid
        )
        ELSE false
    END
ORDER BY 1, 2

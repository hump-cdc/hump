-- snapshot: layout
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET TRANSACTION SNAPSHOT %[1]s;

-- leaves: the table and every inheritance descendant that stores rows
WITH RECURSIVE tree AS (
    SELECT %[2]s::regclass::oid AS oid
    UNION ALL
    SELECT i.inhrelid FROM pg_inherits i JOIN tree t ON i.inhparent = t.oid
)
SELECT
    n.nspname::text AS schema,
    c.relname::text AS name,
    pg_relation_size(c.oid, 'main') / current_setting('block_size')::bigint AS pages
FROM tree t
JOIN pg_class c ON c.oid = t.oid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r'
ORDER BY c.oid;

-- columns: the ones the stream carries, in table order
SELECT a.attname::text AS name, a.atttypid AS type
FROM pg_attribute a
WHERE a.attrelid = %[2]s::regclass
    AND a.attnum > 0
    AND NOT a.attisdropped
    AND a.attgenerated = ''
ORDER BY a.attnum;

COMMIT;

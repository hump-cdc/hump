-- inheritance parent: the hypertable owning a chunk, or the table owning a partition
SELECT
    parent.relnamespace::regnamespace::text AS schema,
    parent.relname::text AS name
FROM pg_inherits i
JOIN pg_class child ON child.oid = i.inhrelid
JOIN pg_class parent ON parent.oid = i.inhparent
JOIN pg_namespace child_ns ON child_ns.oid = child.relnamespace
WHERE child_ns.nspname::text = $1
    AND child.relname::text = $2

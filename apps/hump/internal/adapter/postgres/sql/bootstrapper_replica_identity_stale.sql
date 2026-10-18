SELECT n.nspname::text AS schema, c.relname::text AS name
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE (n.nspname::text, c.relname::text) IN (
        SELECT s, t FROM unnest($1::text[], $2::text[]) AS f(s, t)
    )
    AND c.relreplident <> 'f'

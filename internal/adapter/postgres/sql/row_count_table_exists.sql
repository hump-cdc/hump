-- row count: table exists
SELECT to_regclass($1) IS NOT NULL AS exists

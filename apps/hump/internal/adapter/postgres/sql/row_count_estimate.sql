-- row count: planner estimate
BEGIN READ ONLY;
SET LOCAL statement_timeout = %[1]d;
EXPLAIN (FORMAT JSON) SELECT 1 FROM %[2]s%[3]s;
ROLLBACK;

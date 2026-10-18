-- row count: exact
BEGIN READ ONLY;
SET LOCAL statement_timeout = %[1]d;
SELECT count(*) FROM %[2]s%[3]s;
ROLLBACK;

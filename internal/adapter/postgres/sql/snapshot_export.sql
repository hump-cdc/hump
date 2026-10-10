-- snapshot: export
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET LOCAL lock_timeout = %[1]d;
SET LOCAL idle_in_transaction_session_timeout = 0;
LOCK TABLE %[2]s IN ACCESS SHARE MODE;
SELECT pg_export_snapshot();

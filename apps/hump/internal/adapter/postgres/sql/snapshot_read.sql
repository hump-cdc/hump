-- snapshot: read
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SET TRANSACTION SNAPSHOT %[1]s;
COPY (
    SELECT
        snapshot_row.ctid::text,
        %[2]s
    FROM ONLY %[3]s AS snapshot_row WHERE %[4]s
) TO STDOUT;
COMMIT;

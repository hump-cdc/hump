DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname || '.' || c.relname = %[1]s AND c.relreplident = 'f'
    ) THEN
        ALTER TABLE %[2]s REPLICA IDENTITY FULL;
        RAISE NOTICE 'Table "%%" was updated to REPLICA IDENTITY FULL', %[1]s;
    END IF;
END $$;

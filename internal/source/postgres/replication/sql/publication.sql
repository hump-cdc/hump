DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_publication WHERE pubname = %[1]s) THEN
        ALTER PUBLICATION %[2]s SET %[3]s;
    ELSE
        CREATE PUBLICATION %[2]s FOR %[3]s WITH (publish_via_partition_root = true);
    END IF;

    ALTER PUBLICATION %[2]s SET (publish_via_partition_root = true);
END $$;

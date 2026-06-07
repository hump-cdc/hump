DO $$
DECLARE
    slot_exists BOOLEAN;
    slot_active BOOLEAN;
    slot_active_pid INTEGER;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM pg_replication_slots WHERE slot_name = %[1]s
    ) INTO slot_exists;

    IF NOT slot_exists THEN
        RAISE NOTICE 'Replication slot "%%" does not exist, nothing to drop', %[1]s;
        RETURN;
    END IF;

    SELECT active, active_pid
      INTO slot_active, slot_active_pid
      FROM pg_replication_slots
     WHERE slot_name = %[1]s;

    IF slot_active THEN
        RAISE EXCEPTION 'replication slot "%%" is active (held by pid %%); stop the running hump instance before cleanup',
            %[1]s, slot_active_pid;
    END IF;

    PERFORM pg_drop_replication_slot(%[1]s);
    RAISE NOTICE 'Replication slot "%%" dropped', %[1]s;
END $$;

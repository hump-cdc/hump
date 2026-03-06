DO $$
DECLARE
    slot_exists BOOLEAN;
    plugin_matches BOOLEAN;
    slot_active BOOLEAN;
    slot_active_pid INTEGER;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM pg_replication_slots WHERE slot_name = %[1]s
    ) INTO slot_exists;

    IF slot_exists THEN
        SELECT EXISTS (
            SELECT 1 FROM pg_replication_slots
            WHERE slot_name = %[1]s
            AND plugin = 'pgoutput'
        ) INTO plugin_matches;

        IF NOT plugin_matches THEN
            SELECT active, active_pid
              INTO slot_active, slot_active_pid
              FROM pg_replication_slots
             WHERE slot_name = %[1]s;

            IF slot_active THEN
                RAISE EXCEPTION 'replication slot "%%" is held by pid %% and cannot be recreated with plugin "pgoutput"; terminate the holder or drop the slot manually',
                    %[1]s, slot_active_pid;
            END IF;

            PERFORM pg_drop_replication_slot(%[1]s);
            PERFORM pg_create_logical_replication_slot(%[1]s, 'pgoutput');
            RAISE NOTICE 'Replication slot "%%" recreated with plugin "pgoutput"', %[1]s;
        ELSE
            RAISE NOTICE 'Replication slot "%%" already exists with "pgoutput" plugin, nothing changed', %[1]s;
        END IF;
    ELSE
        PERFORM pg_create_logical_replication_slot(%[1]s, 'pgoutput');
        RAISE NOTICE 'Replication slot "%%" created with plugin "pgoutput"', %[1]s;
    END IF;
END $$;

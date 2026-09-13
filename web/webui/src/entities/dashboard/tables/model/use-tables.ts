import { useMemo } from 'react'

import { useConfig } from '@/entities/config/config'

/** A single row of the tables card: the qualified name plus its configured sink count. */
type Item = { name: string; sinks: number }

/**
 * Tables card rows, sourced from the backend configuration in the order it declares
 * them. `ConfigProvider` resolves the config before the dashboard mounts, so the list
 * is populated on first render.
 */
export const useTables = (): Item[] => {
    const { data: config } = useConfig()

    return useMemo(
        () =>
            (config?.tables ?? []).map((table) => ({
                name: `${table.schema}.${table.name}`,
                sinks: (table.sinks ?? []).length,
            })),
        [config?.tables],
    )
}

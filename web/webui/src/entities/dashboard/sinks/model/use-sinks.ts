import { useMemo } from 'react'

import { useConfig } from '@/entities/config/config/@x/dashboard'

/** A single row of the sinks card. */
type Item = { key: string; type: string; host: string }

/**
 * Sinks card rows, in the order the backend configuration declares them. `ConfigProvider`
 * resolves the config before the dashboard mounts, so the list is populated on first render.
 */
export const useSinks = (): Item[] => {
    const { data: config } = useConfig()

    return useMemo(
        () => (config?.sinks ?? []).map(({ type, host }) => ({ key: `${type} ${host}`, type, host })),
        [config?.sinks],
    )
}

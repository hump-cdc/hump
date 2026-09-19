import { useMemo } from 'react'

import { useLiveQuery } from 'dexie-react-hooks'

import { useConfig } from '@/entities/config/config/@x/dashboard'
import { getFlowSinks } from '@/shared/storage'

/** A single row of the sinks card. */
type Item = { key: string; type: string; host: string; ok: boolean }

/**
 * Share of successful deliveries at or above which a sink is shown as healthy. The card only has
 * a healthy/unhealthy split, so a sink is called out on a noticeable failure share, not one error.
 */
const HEALTHY_SUCCESS_RATE = 99

/** Success rate assumed for a sink with no stored stats, matching the value `seedFlowGraph` writes. */
const UNOBSERVED_SUCCESS_RATE = 100

/** `[sinkType+host]`, the natural key shared by the config sinks and the `flow_sinks` store. */
const sinkKey = (type: string, host: string): string => `${type} ${host}`

/**
 * Sinks card rows. The list and its order come from the backend configuration, resolved before the
 * dashboard mounts; the health flag comes from `successRate` in the `flow_sinks` store.
 */
export const useSinks = (): Item[] => {
    const { data: config } = useConfig()
    const items = useLiveQuery(() => getFlowSinks(), [])

    return useMemo(() => {
        const successRates = new Map((items ?? []).map((sink) => [sinkKey(sink.sinkType, sink.host), sink.successRate]))

        return (config?.sinks ?? []).map(({ type, host }) => ({
            key: sinkKey(type, host),
            type,
            host,
            ok: (successRates.get(sinkKey(type, host)) ?? UNOBSERVED_SUCCESS_RATE) >= HEALTHY_SUCCESS_RATE,
        }))
    }, [config?.sinks, items])
}

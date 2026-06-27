import Dexie, { type Table } from 'dexie'

import { METRIC_SAMPLES_MAX } from '../constants'

/** All rows of one series (or one sink) within a store, ordered oldest → newest. */
export const groupOf = <T, TKey>(table: Table<T, TKey>, index: string, key: string) =>
    table.where(index).between([key, Dexie.minKey], [key, Dexie.maxKey])

/**
 * Trim one group within a bucket store back to `METRIC_SAMPLES_MAX`. The compound primary key
 * orders a group by timestamp, so ranging over it yields the oldest samples first.
 */
export const pruneSamples = async <T, TKey>(table: Table<T, TKey>, index: string, key: string): Promise<void> => {
    const total = await groupOf(table, index, key).count()
    const excess = total - METRIC_SAMPLES_MAX
    if (excess <= 0) return

    const stale = await groupOf(table, index, key).limit(excess).primaryKeys()
    await table.bulkDelete(stale)
}

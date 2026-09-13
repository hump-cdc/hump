import Dexie from 'dexie'

import type { ConfigResponseBodyModel } from '@/shared/api'
import { sha256 } from '@/shared/utils'

import { db } from './db'
import { seedFlowGraph } from './flows/flow-graph'
import { getMeta, saveMeta } from './meta/meta'
import { pruneStaleMetrics } from './metrics/prune'
import { schema } from './schema'

/** Key under which the backend config hash is persisted in the `meta` store. */
const CONFIG_HASH_KEY = 'configHash'

const getConfigHash = (): Promise<string | undefined> => getMeta(CONFIG_HASH_KEY)

const saveConfigHash = (configKey: string): Promise<string> => saveMeta(CONFIG_HASH_KEY, configKey)

/** Derive a stable storage key from the backend config (SHA-256 of its JSON). */
const computeConfigKey = (config: ConfigResponseBodyModel): Promise<string> =>
    sha256(JSON.stringify(config, (_, value) => (typeof value === 'bigint' ? value.toString() : value)))

/** Delete every existing IndexedDB database, dropping stale data from previous sessions. */
const deleteAllDatabases = async (): Promise<void> => {
    if ('databases' in indexedDB) {
        const databases = await indexedDB.databases()
        await Promise.all(databases.map((entry) => (entry.name ? Dexie.delete(entry.name) : Promise.resolve())))
        return
    }
    // Fallback for browsers without `indexedDB.databases()` (e.g. Firefox): drop it by name.
    await Dexie.delete('hump')
}

// Single-flight: bootstrap runs exactly once even under React StrictMode double-mount.
let bootstrap: Promise<void> | null = null

/**
 * Bring the local store online, keyed to a hash of `config`: the database is kept when the stored
 * hash still matches and wiped otherwise, then the flow graph is seeded either way.
 */
export const resetAndInitStorage = (config: ConfigResponseBodyModel): Promise<void> => {
    bootstrap ??= (async () => {
        schema()

        const configKey = await computeConfigKey(config)

        let shouldRecreateDb = true
        try {
            await db.open()
            shouldRecreateDb = (await getConfigHash()) !== configKey
        } catch {
            shouldRecreateDb = true
        }

        if (!shouldRecreateDb) {
            await pruneStaleMetrics(Date.now())
            await seedFlowGraph(config)
            return
        }

        if (db.isOpen()) db.close()

        await deleteAllDatabases()
        await db.open()
        await saveConfigHash(configKey)
        await seedFlowGraph(config)
    })()
    return bootstrap
}

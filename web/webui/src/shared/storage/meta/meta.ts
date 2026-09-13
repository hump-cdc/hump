import { db } from '../db'

/**
 * Read the value stored under `key`, or `undefined` when it was never written. Every reader of
 * the `meta` store goes through here rather than reaching for `db.meta`.
 */
export const getMeta = async (key: string): Promise<string | undefined> => (await db.meta.get(key))?.value

/** Write `value` under `key`, replacing whatever was stored there. Resolves to the key. */
export const saveMeta = (key: string, value: string): Promise<string> => db.meta.put({ key, value })

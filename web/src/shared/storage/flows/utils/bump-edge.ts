import type { IndexableTypeArray, Table } from 'dexie'

/** Add `amount` to the `operationCount` of the row `index` matches, or insert `seed`. */
export const bumpEdge = async <T extends { id?: number; operationCount: number }>(
    table: Table<T, number>,
    index: string,
    key: IndexableTypeArray,
    seed: T,
    amount: number,
): Promise<void> => {
    const existing = await table.where(index).equals(key).first()
    if (!existing) {
        await table.add({ ...seed, operationCount: amount })
        return
    }

    await table.put({ ...existing, operationCount: existing.operationCount + amount })
}

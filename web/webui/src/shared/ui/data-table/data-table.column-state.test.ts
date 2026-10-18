import { afterEach, describe, expect, test } from 'bun:test'

import { loadColumnSizing, parseColumnSizing, saveColumnSizing } from './data-table.column-state'

const KEY = 'hump:data-table:operations:columns'

/** In-memory `Storage`; bun has no `localStorage`, so the tests install one per case. */
class MemoryStorage implements Storage {
    private readonly items = new Map<string, string>()

    get length(): number {
        return this.items.size
    }

    clear = (): void => this.items.clear()

    getItem = (key: string): string | null => this.items.get(key) ?? null

    key = (index: number): string | null => [...this.items.keys()][index] ?? null

    removeItem = (key: string): void => {
        this.items.delete(key)
    }

    setItem = (key: string, value: string): void => {
        this.items.set(key, value)
    }
}

/** Storage whose writes fail, as a full or blocked `localStorage` does. */
class FullStorage extends MemoryStorage {
    override setItem = (): void => {
        throw new Error('QuotaExceededError')
    }
}

const installStorage = (storage: Storage): Storage => {
    globalThis.localStorage = storage
    return storage
}

describe('parseColumnSizing', () => {
    test('keeps the widths of known columns', () => {
        expect(parseColumnSizing({ col_0: 120, col_9: 50, col_1: 'x' }, ['col_0', 'col_1'])).toEqual({ col_0: 120 })
    })

    test('reads the array format the AG Grid table stored, skipping flex entries', () => {
        const stored = [
            { colId: 'col_0', width: 160, flex: null },
            { colId: 'col_1', width: 300, flex: 1 },
        ]

        expect(parseColumnSizing(stored, ['col_0', 'col_1'])).toEqual({ col_0: 160 })
    })

    test('ignores anything else', () => {
        expect(parseColumnSizing('junk', ['col_0'])).toEqual({})
        expect(parseColumnSizing(null, ['col_0'])).toEqual({})
    })
})

describe('loadColumnSizing', () => {
    afterEach(() => {
        Reflect.deleteProperty(globalThis, 'localStorage')
    })

    test('restores the stored widths of known columns', () => {
        installStorage(new MemoryStorage()).setItem(KEY, JSON.stringify({ col_0: 120, col_9: 50 }))

        expect(loadColumnSizing('operations', ['col_0'])).toEqual({ col_0: 120 })
    })

    test('starts empty when nothing is stored', () => {
        installStorage(new MemoryStorage())

        expect(loadColumnSizing('operations', ['col_0'])).toEqual({})
    })

    test('starts empty when the stored value is not JSON or storage is unavailable', () => {
        expect(loadColumnSizing('operations', ['col_0'])).toEqual({})

        installStorage(new MemoryStorage()).setItem(KEY, '{broken')
        expect(loadColumnSizing('operations', ['col_0'])).toEqual({})
    })
})

describe('saveColumnSizing', () => {
    afterEach(() => {
        Reflect.deleteProperty(globalThis, 'localStorage')
    })

    test('stores the widths under the table key', () => {
        const storage = installStorage(new MemoryStorage())
        saveColumnSizing('operations', { col_0: 120 })

        expect(storage.getItem(KEY)).toBe('{"col_0":120}')
        expect(loadColumnSizing('operations', ['col_0'])).toEqual({ col_0: 120 })
    })

    test('ignores a failing or missing storage', () => {
        expect(() => saveColumnSizing('operations', { col_0: 120 })).not.toThrow()

        installStorage(new FullStorage())
        expect(() => saveColumnSizing('operations', { col_0: 120 })).not.toThrow()
    })
})

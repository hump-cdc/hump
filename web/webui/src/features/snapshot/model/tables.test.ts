import { afterAll, beforeEach, describe, expect, test } from 'bun:test'

const STORAGE_KEY = 'snapshot-tables'

/** In-memory `Storage`; bun has no `localStorage`, and the store binds to it when its module loads. */
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

const storage = new MemoryStorage()
storage.setItem(
    STORAGE_KEY,
    JSON.stringify({ state: { selectedIds: ['public.users'], wheres: { 'public.users': 'id > 1' } }, version: 1 }),
)
globalThis.localStorage = storage

// Imported after the storage is in place: the persist middleware reads it on creation.
const { useTables } = await import('./tables')

const readPersisted = (): unknown => JSON.parse(storage.getItem(STORAGE_KEY) ?? 'null')

afterAll(() => {
    Reflect.deleteProperty(globalThis, 'localStorage')
})

describe('useTables persistence', () => {
    test('restores the selection as a set and the filters on load', () => {
        const { selectedIds, wheres, filter } = useTables.getState()

        expect([...selectedIds]).toEqual(['public.users'])
        expect(wheres).toEqual({ 'public.users': 'id > 1' })
        expect(filter).toBe('')
    })

    test('stores the selection as an array and leaves the name filter out', () => {
        useTables.getState().setFilter('ord')
        useTables.getState().toggle('public.orders')

        expect(readPersisted()).toEqual({
            state: { selectedIds: ['public.users', 'public.orders'], wheres: { 'public.users': 'id > 1' } },
            version: 1,
        })
    })

    test('falls back to an empty selection when nothing usable is stored', async () => {
        storage.setItem(STORAGE_KEY, JSON.stringify({ state: {}, version: 1 }))
        await useTables.persist.rehydrate()

        expect(useTables.getState().selectedIds.size).toBe(0)
        expect(useTables.getState().wheres).toEqual({})
    })
})

describe('useTables', () => {
    beforeEach(() => {
        useTables.setState({ filter: '', selectedIds: new Set(), wheres: {} })
    })

    test('sets the name filter', () => {
        useTables.getState().setFilter('orders')

        expect(useTables.getState().filter).toBe('orders')
    })

    test('toggles a single table', () => {
        useTables.getState().toggle('public.orders')
        expect([...useTables.getState().selectedIds]).toEqual(['public.orders'])

        useTables.getState().toggle('public.orders')
        expect(useTables.getState().selectedIds.size).toBe(0)
    })

    test('selects all given tables unless every one is already selected', () => {
        useTables.getState().toggle('public.users')

        useTables.getState().toggleAll(['public.users', 'public.orders'])
        expect([...useTables.getState().selectedIds]).toEqual(['public.users', 'public.orders'])

        useTables.getState().toggleAll(['public.users', 'public.orders'])
        expect(useTables.getState().selectedIds.size).toBe(0)
    })

    test('keeps tables outside the given ids selected', () => {
        useTables.getState().toggle('audit.log')
        useTables.getState().toggleAll(['public.users'])
        useTables.getState().toggleAll(['public.users'])

        expect([...useTables.getState().selectedIds]).toEqual(['audit.log'])
    })

    test('sets a filter per table, keeping the state when it is unchanged', () => {
        useTables.getState().setWhere('public.users', 'id > 1')
        const state = useTables.getState()

        useTables.getState().setWhere('public.users', 'id > 1')
        expect(useTables.getState()).toBe(state)

        useTables.getState().setWhere('public.users', '')
        expect(useTables.getState().wheres).toEqual({ 'public.users': '' })
    })
})

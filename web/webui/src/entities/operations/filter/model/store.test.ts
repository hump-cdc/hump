import { afterEach, beforeEach, describe, expect, type Mock, spyOn, test } from 'bun:test'

import { defaultFilterValues, useOperationsFilter } from './store'

// Bun has no `localStorage`, so the persist middleware warns on every `set`; the warning is muted.
let warn: Mock<typeof console.warn>

beforeEach(() => {
    warn = spyOn(console, 'warn').mockImplementation(() => {})
    useOperationsFilter.setState(useOperationsFilter.getInitialState(), true)
})

afterEach(() => {
    warn.mockRestore()
})

describe('useOperationsFilter', () => {
    test('starts from the filter that allows every option', () => {
        const { sinks, targets, tables, operations, statuses, search } = useOperationsFilter.getState()

        expect({ sinks, targets, tables, operations, statuses, search }).toEqual(defaultFilterValues)
        expect(defaultFilterValues.operations).toEqual(['insert', 'update', 'delete', 'truncate'])
        expect(defaultFilterValues.statuses).toEqual(['delivered', 'failed'])
    })

    test('each setter overwrites its own facet only', () => {
        const state = useOperationsFilter.getState()

        state.setSinks(['kafka'])
        state.setTargets(['users'])
        state.setTables(['public.users'])
        state.setOperations(['insert'])
        state.setStatuses(['failed'])
        state.setSearch('timeout')

        expect(useOperationsFilter.getState()).toMatchObject({
            sinks: ['kafka'],
            targets: ['users'],
            tables: ['public.users'],
            operations: ['insert'],
            statuses: ['failed'],
            search: 'timeout',
        })
    })

    test('a facet set back to null allows every option again', () => {
        useOperationsFilter.getState().setSinks(['kafka'])
        useOperationsFilter.getState().setSinks(null)

        expect(useOperationsFilter.getState().sinks).toBeNull()
    })

    test('apply overwrites the given facets and keeps the others', () => {
        useOperationsFilter.getState().setSearch('timeout')
        useOperationsFilter.getState().apply({ tables: ['public.users'], statuses: ['failed'] })

        expect(useOperationsFilter.getState()).toMatchObject({
            ...defaultFilterValues,
            tables: ['public.users'],
            statuses: ['failed'],
            search: 'timeout',
        })
    })
})

import { beforeEach, describe, expect, test } from 'bun:test'

import type { Operation } from '@/entities/operations/list/@x/details'

import { useOperationDetailsStore } from './store'

const OPERATION: Operation = {
    id: '42',
    time: new Date(2026, 9, 17, 14, 3, 5),
    table: 'public.users',
    op: 'insert',
    sink: 'kafka',
    sinkHost: 'localhost:9092',
    target: 'users',
    status: 'delivered',
    attempts: 1,
    durationMs: 12,
    payload: '{}',
    payloadSize: 2,
    metadata: {},
}

beforeEach(() => {
    useOperationDetailsStore.setState(useOperationDetailsStore.getInitialState(), true)
})

describe('useOperationDetailsStore', () => {
    test('starts with nothing selected', () => {
        expect(useOperationDetailsStore.getState().selected).toBeNull()
    })

    test('select keeps the whole row', () => {
        useOperationDetailsStore.getState().select(OPERATION)

        expect(useOperationDetailsStore.getState().selected).toBe(OPERATION)
    })

    test('clear drops the selection', () => {
        useOperationDetailsStore.getState().select(OPERATION)
        useOperationDetailsStore.getState().clear()

        expect(useOperationDetailsStore.getState().selected).toBeNull()
    })
})

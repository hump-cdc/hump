import { beforeEach, describe, expect, test } from 'bun:test'

import { useFlowInspectorStore } from './store'
import './types'

const SINK = { type: 'kafka', host: 'localhost:9092' }

beforeEach(() => {
    useFlowInspectorStore.setState(useFlowInspectorStore.getInitialState(), true)
})

describe('useFlowInspectorStore', () => {
    test('starts with nothing selected', () => {
        expect(useFlowInspectorStore.getState().selected).toBeNull()
    })

    test('tags each selection with its kind', () => {
        const { selectTable, selectSink, selectTarget, selectEdge } = useFlowInspectorStore.getState()

        selectTable({ id: 'public.users', name: 'users' })
        expect(useFlowInspectorStore.getState().selected).toEqual({ kind: 'table', id: 'public.users', name: 'users' })

        selectSink({ id: 'kafka|localhost:9092', name: 'kafka', ...SINK })
        expect(useFlowInspectorStore.getState().selected).toEqual({
            kind: 'sink',
            id: 'kafka|localhost:9092',
            name: 'kafka',
            ...SINK,
        })

        selectTarget({ id: 'kafka|localhost:9092/users', name: 'users', sinkLabel: 'kafka', ...SINK })
        expect(useFlowInspectorStore.getState().selected).toEqual({
            kind: 'target',
            id: 'kafka|localhost:9092/users',
            name: 'users',
            sinkLabel: 'kafka',
            ...SINK,
        })

        selectEdge({ id: 'edge', sourceLabel: 'users', targetLabel: 'kafka', table: 'public.users', ...SINK })
        expect(useFlowInspectorStore.getState().selected).toEqual({
            kind: 'edge',
            id: 'edge',
            sourceLabel: 'users',
            targetLabel: 'kafka',
            table: 'public.users',
            ...SINK,
        })
    })

    test('clear drops the selection', () => {
        useFlowInspectorStore.getState().selectTable({ id: 'public.users', name: 'users' })
        useFlowInspectorStore.getState().clear()

        expect(useFlowInspectorStore.getState().selected).toBeNull()
    })
})

import { createRef } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render, renderHook } from '@testing-library/react'
import { type Edge, ReactFlowProvider } from '@xyflow/react'

import { TooltipProvider } from '@/shared/ui/tooltip'

import { useDetailedEdgeAnimation } from './flow-detailed.helpers'
import {
    FlowDetailed,
    FlowDetailedGhost,
    type FlowDetailedRef,
    FlowDetailedSink,
    FlowDetailedTable,
    type SinkTarget,
    sinkTargetKey,
    type TableNodeData,
} from './index'
import { TargetRow } from './node.sink.target'

// Type-only module: loaded for its side effects so the coverage report lists it.
import './flow-detailed.types'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

const ops: TableNodeData['ops'] = {
    insert: { count: 1 },
    update: { count: 0 },
    delete: { count: 0 },
    truncate: { count: 0 },
}

const topic: SinkTarget = {
    id: 'topic',
    name: 'orders',
    kind: 'topic',
    count: 5,
    p95: 12,
    successRate: 99,
    health: 'ok',
}
const idle: SinkTarget = { id: 'idle', name: 'archive', kind: 'topic', count: 0, health: 'error' }

const edges: Edge[] = [
    { id: 'e1', source: 't', target: 's', targetHandle: 'topic' },
    { id: 'e2', source: 't', target: 'g' },
]

type Handlers = {
    onTableNodeSelect: ReturnType<typeof mock>
    onSinkNodeSelect: ReturnType<typeof mock>
    onSinkTargetSelect: ReturnType<typeof mock>
    onClearSelection: ReturnType<typeof mock>
}

/** Renders a canvas with one table, one sink (two targets), one ghost and an overlay. */
const mount = (selected?: string) => {
    const handlers: Handlers = {
        onTableNodeSelect: mock(),
        onSinkNodeSelect: mock(),
        onSinkTargetSelect: mock(),
        onClearSelection: mock(),
    }
    const view = render(
        <TooltipProvider>
            <FlowDetailed ref={createRef<FlowDetailedRef>()} selected={selected} edges={edges} {...handlers}>
                <FlowDetailedTable id="t" name="public.orders" ops={ops} />
                <FlowDetailedSink
                    id="s"
                    name="Kafka"
                    type="kafka"
                    host="broker:9092"
                    count={5}
                    targets={[topic, idle]}
                />
                <FlowDetailedSink id="s2" name="Webhook" type="http" count={0} targets={[]} />
                <FlowDetailedGhost id="g" type="clickhouse" />
                <div>overlay</div>
            </FlowDetailed>
        </TooltipProvider>,
    )

    return { ...view, ...handlers }
}

/** The React Flow node wrapper with `id`. */
const nodeById = (container: HTMLElement, id: string): Element => {
    const node = container.querySelector(`[data-id="${id}"]`)
    if (!node) throw new Error(`node ${id} not rendered`)
    return node
}

describe('FlowDetailed', () => {
    test('lays out tables, sinks with their targets and ghosts', () => {
        const { getByText, queryByText } = mount()

        expect(getByText('public.orders')).toBeDefined()
        expect(getByText('broker:9092')).toBeDefined()
        expect(getByText('orders')).toBeDefined()
        expect(getByText('archive')).toBeDefined()
        expect(getByText('clickhouse')).toBeDefined()
        expect(getByText('TABLES')).toBeDefined()
        expect(getByText('SINKS')).toBeDefined()
        expect(getByText('overlay')).toBeDefined()
        expect(queryByText('Webhook')).not.toBeNull()
    })

    test('selects table and sink nodes', () => {
        const { container, onTableNodeSelect, onSinkNodeSelect } = mount()

        fireEvent.click(nodeById(container, 't'))
        fireEvent.click(nodeById(container, 's'))

        expect(onTableNodeSelect).toHaveBeenCalledWith('t', { name: 'public.orders', ops })
        expect(onSinkNodeSelect).toHaveBeenCalledWith('s', expect.objectContaining({ name: 'Kafka' }))
    })

    test('selects a target row without selecting its sink', () => {
        const { getByText, onSinkTargetSelect, onSinkNodeSelect } = mount()

        fireEvent.click(getByText('orders'))

        expect(onSinkTargetSelect).toHaveBeenCalledWith('s', topic)
        expect(onSinkNodeSelect).not.toHaveBeenCalled()
    })

    test('clears the selection when the selected target is clicked again', () => {
        const { container, getByText, onClearSelection, onSinkTargetSelect } = mount(sinkTargetKey('s', 'topic'))

        expect(container.querySelector('.selected-target')).not.toBeNull()

        fireEvent.click(getByText('orders'))

        expect(onClearSelection).toHaveBeenCalledTimes(1)
        expect(onSinkTargetSelect).not.toHaveBeenCalled()
    })

    test('clears the selection on a pane click', () => {
        const { container, onClearSelection } = mount('t')
        const pane = container.querySelector('.react-flow__pane')
        if (!pane) throw new Error('pane not rendered')

        fireEvent.click(pane)

        expect(onClearSelection).toHaveBeenCalledTimes(1)
    })

    test('renders nothing for the config elements themselves', () => {
        const { container } = render(
            <>
                <FlowDetailedGhost id="g" type="s3" />
                <FlowDetailedSink id="s" name="Kafka" type="kafka" count={0} targets={[]} />
            </>,
        )

        expect(container.innerHTML).toBe('')
    })
})

describe('useDetailedEdgeAnimation', () => {
    test('animates the edge into a target or a ghost and ignores an unknown route', () => {
        const ref = createRef<FlowDetailedRef>()
        const { result } = renderHook(() => useDetailedEdgeAnimation(edges, ref))

        act(() => {
            ref.current?.animate('t', 's', 'topic', 10)
            ref.current?.animate('t', 'g')
            ref.current?.animate('t', 's', 'missing')
        })

        expect([...result.current.keys()]).toEqual(['e1', 'e2'])
    })
})

describe('TargetRow', () => {
    test('is inert outside a FlowDetailed canvas', () => {
        const { getByText, container } = render(
            <ReactFlowProvider>
                <TargetRow sinkId="s" target={idle} />
            </ReactFlowProvider>,
        )

        fireEvent.click(getByText('archive'))

        expect(container.querySelector('.selected-target')).toBeNull()
    })
})

import { createRef } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render, renderHook } from '@testing-library/react'
import type { Edge } from '@xyflow/react'

import { TooltipProvider } from '@/shared/ui/tooltip'

import { useSummaryEdgeAnimation } from './flow-summary.helpers'
import { FlowSummary, type FlowSummaryRef, FlowSummarySink, FlowSummaryTable, type TableNodeData } from './index'

// Type-only module: loaded for its side effects so the coverage report lists it.
import './flow-summary.types'

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

const edges: Edge[] = [{ id: 'e1', source: 't', target: 's' }]

/** Renders a canvas with one table, three sinks (healthy, failing, idle) and an overlay. */
const mount = (selected?: string) => {
    const handlers = { onTableNodeSelect: mock(), onSinkNodeSelect: mock(), onClearSelection: mock() }
    const view = render(
        <TooltipProvider>
            <FlowSummary ref={createRef<FlowSummaryRef>()} selected={selected} edges={edges} {...handlers}>
                <FlowSummaryTable id="t" name="public.orders" ops={ops} />
                <FlowSummarySink
                    id="s"
                    name="Kafka"
                    type="kafka"
                    host="broker:9092"
                    operations={5}
                    p95={12}
                    successRate={100}
                    health="ok"
                />
                <FlowSummarySink id="s2" name="Webhook" type="http" operations={3} successRate={50} health="error" />
                <FlowSummarySink id="s3" name="Idle" type="nats-core" operations={0} health="ok" />
                <div>overlay</div>
            </FlowSummary>
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

describe('FlowSummary', () => {
    test('lays out tables and sinks with their metrics', () => {
        const { getByText, getAllByText } = mount()

        expect(getByText('public.orders')).toBeDefined()
        expect(getByText('broker:9092')).toBeDefined()
        expect(getByText('Webhook')).toBeDefined()
        expect(getByText('Idle')).toBeDefined()
        expect(getAllByText('OPERATIONS')).toHaveLength(3)
        expect(getByText('TABLES')).toBeDefined()
        expect(getByText('SINKS')).toBeDefined()
        expect(getByText('overlay')).toBeDefined()
    })

    test('tones the success rate by health', () => {
        const { container } = mount()
        const success = (id: string): string | undefined =>
            nodeById(container, id).querySelector('.grid > :last-child [data-slot=tooltip-trigger]')?.className

        expect(success('s')).toContain('text-success')
        expect(success('s2')).toContain('text-danger')
        expect(success('s3')).toContain('text-ink-subtle')
    })

    test('selects table and sink nodes', () => {
        const { container, onTableNodeSelect, onSinkNodeSelect } = mount()

        fireEvent.click(nodeById(container, 't'))
        fireEvent.click(nodeById(container, 's'))

        expect(onTableNodeSelect).toHaveBeenCalledWith('t', { name: 'public.orders', ops })
        expect(onSinkNodeSelect).toHaveBeenCalledWith('s', expect.objectContaining({ name: 'Kafka' }))
    })

    test('clears the selection when the selected node is clicked again', () => {
        const { container, onClearSelection, onSinkNodeSelect } = mount('s')

        fireEvent.click(nodeById(container, 's'))

        expect(onClearSelection).toHaveBeenCalledTimes(1)
        expect(onSinkNodeSelect).not.toHaveBeenCalled()
    })

    test('renders nothing for the sink config element itself', () => {
        const { container } = render(<FlowSummarySink id="s" name="Kafka" type="kafka" operations={0} health="ok" />)
        expect(container.innerHTML).toBe('')
    })
})

describe('useSummaryEdgeAnimation', () => {
    test('animates an existing route and ignores an unknown one', () => {
        const ref = createRef<FlowSummaryRef>()
        const { result } = renderHook(() => useSummaryEdgeAnimation(edges, ref))

        act(() => {
            ref.current?.animate('t', 's', 10)
            ref.current?.animate('t', 'missing')
        })

        expect([...result.current.keys()]).toEqual(['e1'])
    })
})

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'
import type { Edge, Node, NodeTypes } from '@xyflow/react'

import { TooltipProvider } from '@/shared/ui/tooltip'

import { Flow, FlowControlButton, LabelNodeView, SinkIcon, SinkIconTile, TableNode, TableNodeView } from './index'

// Type-only modules: loaded for their side effects so the coverage report lists them.
import './flow.types'
import './node'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

const nodeTypes: NodeTypes = { table: TableNodeView, label: LabelNodeView }

const nodes: Node[] = [
    {
        id: 't',
        type: 'table',
        position: { x: 0, y: 0 },
        data: {
            name: 'public.users',
            ops: { insert: { count: 1, p95: 4 }, update: { count: 2 }, delete: { count: 3 }, truncate: { count: 4 } },
        },
    },
    { id: 'l', type: 'label', position: { x: 0, y: -36 }, data: { text: 'TABLES' } },
]

const edges: Edge[] = []

describe('Flow', () => {
    test('renders table and label nodes, extra controls and overlay children', () => {
        const { container, getByText, getByLabelText } = render(
            <TooltipProvider>
                <Flow
                    nodes={nodes}
                    edges={edges}
                    nodeTypes={nodeTypes}
                    revealId="t"
                    controls={<FlowControlButton aria-label="Extra" />}
                >
                    <div>overlay</div>
                </Flow>
            </TooltipProvider>,
        )

        expect(getByText('public.users')).toBeDefined()
        expect(getByText('TABLES')).toBeDefined()
        expect(getByText('INS')).toBeDefined()
        expect(getByText('overlay')).toBeDefined()
        expect(getByLabelText('Extra')).toBeDefined()
        expect(container.querySelector('.react-flow__node-table')).not.toBeNull()
    })

    test('reports viewport moves to onMoveStart', async () => {
        const onMoveStart = mock()
        const { container } = render(
            <TooltipProvider>
                <Flow nodes={nodes} edges={edges} nodeTypes={nodeTypes} onMoveStart={onMoveStart} />
            </TooltipProvider>,
        )
        const pane = container.querySelector('.react-flow__renderer')
        if (!pane) throw new Error('pane not rendered')

        fireEvent.wheel(pane, { deltaY: 100, ctrlKey: true })
        await act(() => Bun.sleep(10))

        expect(onMoveStart).toHaveBeenCalled()
        expect(onMoveStart.mock.calls[0][0]).toBeTruthy()
    })

    test('renders without a move handler', () => {
        const { container } = render(<Flow nodes={[]} edges={edges} />)
        expect(container.querySelector('.react-flow')).not.toBeNull()
    })
})

describe('TableNode', () => {
    test('is a config element that renders nothing', () => {
        expect(TableNode.flowNodeType).toBe('table')
        const { container } = render(
            <TableNode
                id="t"
                name="x"
                ops={{ insert: { count: 0 }, update: { count: 0 }, delete: { count: 0 }, truncate: { count: 0 } }}
            />,
        )
        expect(container.innerHTML).toBe('')
    })
})

describe('SinkIcon', () => {
    test('falls back to a generic icon for an unknown type', () => {
        const known = render(<SinkIcon type="kafka" />).container.innerHTML
        const unknown = render(<SinkIcon type="unknown" />).container.innerHTML

        expect(unknown).toContain('lucide-plug')
        expect(known).not.toContain('lucide-plug')
    })

    test('wraps the icon in a tile', () => {
        const { container } = render(<SinkIconTile type="http" className="custom" />)
        expect(container.firstElementChild?.className).toContain('custom')
        expect(container.innerHTML).toContain('lucide-globe')
    })
})

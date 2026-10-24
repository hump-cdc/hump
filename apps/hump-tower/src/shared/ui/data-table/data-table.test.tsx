import type { FC } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'

import './data-table.types'
import { DataTable, type DataTableColumn, type DataTableProps } from './index'

type Row = { id: string; name: string; size?: number }

/** Body size the stubbed layout reports: room for ten 40px rows. */
const VIEWPORT = { width: 800, height: 400 }

const rows: Row[] = Array.from({ length: 5 }, (_, index) => ({ id: `r${index}`, name: `Row ${index}` }))

const columns: DataTableColumn<Row>[] = [
    { field: 'name', header: 'Name', width: 200, headerClass: 'head', cellClass: 'cell' },
    { header: 'Upper', flex: 1, render: (row) => row.name.toUpperCase() },
    { field: 'size', header: 'Size', width: 100, resizable: false },
    { header: 'Blank', width: 80 },
]

const Detail: FC<{ row: Row }> = ({ row }) => <div data-testid="detail">detail {row.id}</div>

/** Layout getters happy-dom leaves at zero, replaced while the tests run. */
const stubbedLayout = ['clientWidth', 'clientHeight', 'offsetWidth', 'offsetHeight']

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
    stubLayout()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(() => {
    cleanup()
    localStorage.clear()
})

describe('DataTable', () => {
    test('shows the empty message when there are no rows', () => {
        const view = mount({ rowCount: 0, emptyMessage: 'Nothing here', height: 300 })

        expect(view.getByText('Nothing here')).toBeDefined()
        expect(view.container.querySelector('[data-slot="data-table-row"]')).toBeNull()
    })

    test('shows skeleton rows while the row count is loading', () => {
        const view = mount({ rowCount: 0, isLoading: true })

        expect(view.container.querySelectorAll('[data-slot="data-table-row"]').length).toBeGreaterThan(0)
        expect(view.container.querySelector('.animate-pulse')).not.toBeNull()
    })

    test('renders headers and loaded rows, skeletons for unloaded ones, and reports the range', () => {
        const onVisibleRangeChange = mock()
        const view = mount({
            rowCount: 6,
            getRow: (index) => rows[index],
            onVisibleRangeChange,
        })

        expect(view.getByText('Name').parentElement?.className).toContain('head')
        expect(view.getByText('Row 0').className).toContain('cell')
        expect(view.getByText('ROW 1')).toBeDefined()
        expect(view.container.querySelector('.animate-pulse')).not.toBeNull()
        expect(onVisibleRangeChange).toHaveBeenCalledWith({ start: 0, end: 5 })
    })

    test('selects a clicked row and clears on a click outside the rows', () => {
        const onRowSelect = mock()
        const onClearSelection = mock()
        const view = mount({ onRowSelect, onClearSelection, selectedRowId: 'r0' })

        fireEvent.click(view.getByText('Row 2'))
        expect(onRowSelect).toHaveBeenCalledWith('r2', rows[2])
        expect(onClearSelection).not.toHaveBeenCalled()

        fireEvent.click(body(view.container))
        expect(onClearSelection).toHaveBeenCalledTimes(1)
    })

    test('expands and collapses a row through its chevron without selecting it', () => {
        const onRowSelect = mock()
        const view = mount({
            getDetail: Detail,
            detailHeight: (row) => (row.id === 'r0' ? 120 : 80),
            isRowExpandable: (row) => row.id !== 'r4',
            onRowSelect,
        })

        expect(view.getAllByLabelText('Expand row')).toHaveLength(4)

        const chevron = view.getAllByLabelText('Expand row')[0]
        fireEvent.mouseDown(chevron)
        fireEvent.click(chevron)
        expect(view.getByTestId('detail').textContent).toBe('detail r0')
        expect(onRowSelect).not.toHaveBeenCalled()

        fireEvent.click(view.getByLabelText('Collapse row'))
        expect(view.queryByTestId('detail')).toBeNull()
    })

    test('moves the selection with the arrow keys and toggles it with Space', () => {
        const onRowSelect = mock()
        const view = mount({ getDetail: Detail, onRowSelect })
        const scroller = body(view.container)

        // No selection yet: start from the first row on screen.
        fireEvent.keyDown(scroller, { key: 'ArrowDown' })
        expect(onRowSelect).toHaveBeenLastCalledWith('r0', rows[0])

        // Space without a selection does nothing.
        fireEvent.keyDown(scroller, { key: ' ' })
        expect(view.queryByTestId('detail')).toBeNull()

        view.rerender(props({ getDetail: Detail, onRowSelect, selectedRowId: 'r0' }))
        fireEvent.keyDown(scroller, { key: 'ArrowDown' })
        expect(onRowSelect).toHaveBeenLastCalledWith('r1', rows[1])

        view.rerender(props({ getDetail: Detail, onRowSelect, selectedRowId: 'r1' }))
        fireEvent.keyDown(scroller, { key: 'ArrowUp' })
        expect(onRowSelect).toHaveBeenLastCalledWith('r0', rows[0])

        fireEvent.keyDown(scroller, { key: ' ' })
        expect(view.getByTestId('detail').textContent).toBe('detail r1')

        // Other keys are ignored.
        fireEvent.keyDown(scroller, { key: 'Enter' })
        expect(onRowSelect).toHaveBeenCalledTimes(3)
    })

    test('finds a selection among the rendered rows and stops at the edges', () => {
        const onRowSelect = mock()
        const view = mount({ onRowSelect, selectedRowId: 'r4' })
        const scroller = body(view.container)

        fireEvent.keyDown(scroller, { key: 'ArrowDown' })
        expect(onRowSelect).not.toHaveBeenCalled()

        view.rerender(props({ onRowSelect, selectedRowId: 'r0' }))
        fireEvent.keyDown(scroller, { key: 'ArrowUp' })
        expect(onRowSelect).not.toHaveBeenCalled()
    })

    test('skips an unloaded row on the arrow keys', () => {
        const onRowSelect = mock()
        const view = mount({ rowCount: 6, getRow: (index) => rows[index], onRowSelect, selectedRowId: 'r4' })

        fireEvent.keyDown(body(view.container), { key: 'ArrowDown' })
        expect(onRowSelect).not.toHaveBeenCalled()
    })

    test('leaves keys alone without row selection or when typed inside a row', () => {
        const view = mount({ getDetail: Detail })
        fireEvent.keyDown(body(view.container), { key: 'ArrowDown' })

        const onRowSelect = mock()
        view.rerender(props({ onRowSelect }))
        fireEvent.keyDown(view.getByText('Row 0'), { key: 'ArrowDown' })
        expect(onRowSelect).not.toHaveBeenCalled()
    })

    test('ignores clicks on the body without a clear handler and mirrors horizontal scroll onto the header', () => {
        const view = mount({})
        const scroller = body(view.container)
        fireEvent.click(scroller)

        scroller.scrollLeft = 30
        fireEvent.scroll(scroller)
        const header = scroller.parentElement?.previousElementSibling

        expect(header?.scrollLeft).toBe(30)
    })

    test('persists a resized column under its id and restores it on mount', () => {
        localStorage.setItem('hump:data-table:t1:columns', JSON.stringify({ col_0: 260 }))
        const view = mount({ id: 't1' })

        expect(cellWidth(view.getByText('Name'))).toBe('260px')

        const grips = view.container.querySelectorAll('[aria-hidden="true"]')
        const grip = grips[0]
        expect(grip.className).toContain('cursor-ew-resize')
        expect(grips[1].className).toContain('cursor-default')

        // Separate events, so the table renders the drag before it ends.
        fireEvent.mouseDown(grip, { clientX: 100 })
        fireEvent.mouseMove(document, { clientX: 150 })
        fireEvent.mouseUp(document, { clientX: 150 })

        const saved = JSON.parse(localStorage.getItem('hump:data-table:t1:columns') ?? '{}')
        expect(saved.col_0).toBe(310)
    })

    test('keeps resizes in memory when the table has no id', () => {
        const view = mount({})
        const grip = view.container.querySelectorAll('[aria-hidden="true"]')[0]

        act(() => {
            fireEvent.mouseDown(grip, { clientX: 100 })
            fireEvent.mouseMove(document, { clientX: 140 })
            fireEvent.mouseUp(document, { clientX: 140 })
        })

        expect(cellWidth(view.getByText('Name'))).toBe('240px')
        expect(localStorage.length).toBe(0)
    })
})

/** Table props over the five test rows, with `overrides` on top. */
const props = (overrides: Partial<DataTableProps<Row>>) => (
    <DataTable<Row>
        columns={columns}
        getRowId={(row) => row.id}
        rowCount={rows.length}
        getRow={(index) => rows[index]}
        rowHeight={40}
        overscan={2}
        {...overrides}
    />
)

/** Renders the table with `overrides` applied to the default props. */
const mount = (overrides: Partial<DataTableProps<Row>>) => render(props(overrides))

/** The scrolling body, the table's keyboard and click target. */
const body = (container: HTMLElement): HTMLElement => {
    const element = container.querySelector<HTMLElement>('.overflow-auto')
    if (!element) throw new Error('table body not rendered')
    return element
}

/** Inline width of the header cell holding `label`. */
const cellWidth = (label: HTMLElement): string | undefined => label.parentElement?.style.width

/** Gives every element the viewport's size, which the virtualizer and flex columns read. */
const stubLayout = (): void => {
    for (const key of stubbedLayout) {
        Object.defineProperty(HTMLElement.prototype, key, {
            configurable: true,
            get: () => (key.endsWith('Width') ? VIEWPORT.width : VIEWPORT.height),
        })
    }
    Element.prototype.getBoundingClientRect = () => new DOMRect(0, 0, VIEWPORT.width, VIEWPORT.height)
}

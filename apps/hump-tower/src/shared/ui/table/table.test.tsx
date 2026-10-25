import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, fireEvent, render } from '@testing-library/react'

import { Table, TableRow, type TableRowColumn, TableRowHeader } from './index'

// Radix picks its layout effect when first imported, as a no-op without a DOM; bun shares modules
// across test files, so point it at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))

type Row = { name: string; host: string; status: string }

const row: Row = { name: 'orders', host: 'kafka:9092', status: 'healthy' }

const columns: TableRowColumn<Row>[] = [
    {
        key: 'name',
        header: 'Name',
        className: 'name-col',
        cell: (r) => <span>{`cell:${r.name}`}</span>,
        mobile: { zone: 'title', render: (r) => <span>{`title:${r.name}`}</span> },
    },
    {
        key: 'host',
        header: 'Host',
        cell: (r) => <span>{`cell:${r.host}`}</span>,
        mobile: { zone: 'meta', className: 'meta-wrap', render: (r) => <span>{`meta:${r.host}`}</span> },
    },
    { key: 'status', cell: (r) => <span>{`cell:${r.status}`}</span> },
]

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('Table', () => {
    test('renders a framed table with a card header, toolbar and rows', () => {
        const view = render(
            <Table
                columns={columns}
                variant="bordered"
                header="Flows"
                headerAside={<span>3 total</span>}
                toolbar={<span>Toolbar</span>}
                className="x"
            >
                <TableRow row={row} columns={columns} mobileZoneClasses={{ title: 'title-x' }} />
            </Table>,
        )

        expect(view.getByText('Flows')).toBeDefined()
        expect(view.getByText('3 total')).toBeDefined()
        expect(view.getByText('Toolbar').parentElement?.className).toContain('p-4')
        expect(view.getByText('Name').className).toContain('name-col')
        expect(view.getByText('cell:orders').parentElement?.className).toBe('name-col')
        expect(view.getByText('title:orders').parentElement?.className).toContain('title-x')
        expect(view.getByText('meta:kafka:9092').parentElement?.className).toBe('meta-wrap')
        expect(view.getByText('cell:healthy')).toBeDefined()
        expect(view.container.querySelector('.x')).not.toBeNull()
    })

    test('renders an unframed table with its aside and toolbar', () => {
        const view = render(
            <Table columns={columns} headerAside={<span>Aside</span>} toolbar={<span>Toolbar</span>}>
                <TableRow row={row} columns={columns} />
            </Table>,
        )

        expect(view.getByText('Aside').parentElement?.className).toContain('gap-2')
        expect(view.getByText('Toolbar').parentElement?.className).toContain('py-4')
    })

    test('renders without a header row when neither header nor aside is given', () => {
        const view = render(
            <Table columns={columns}>
                <span>Body</span>
            </Table>,
        )

        expect(view.container.firstElementChild?.children).toHaveLength(2)
    })

    test('renders selectable rows that toggle from their label', () => {
        const onToggle = mock(() => {})
        const view = render(
            <Table columns={columns} selectable>
                <TableRow row={row} columns={columns} selectable={{ id: 'row-1', checked: true, onToggle }} />
            </Table>,
        )

        expect(view.getByText('cell:orders').closest('label')?.className).toContain('bg-primary-soft/40')

        fireEvent.click(view.getByRole('checkbox'))
        expect(onToggle).toHaveBeenCalledTimes(1)
    })
})

describe('TableRowHeader', () => {
    test('reserves the checkbox column when selectable', () => {
        const view = render(<TableRowHeader columns={columns} selectable />)

        expect(view.container.firstElementChild?.firstElementChild?.className).toContain('size-4.5')
    })
})

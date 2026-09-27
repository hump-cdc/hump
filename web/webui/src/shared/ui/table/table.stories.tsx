import { useState } from 'react'

import type { Meta, StoryObj } from '@storybook/react'

import { Badge } from '@/shared/ui/badge'

import { Table } from './table'
import type { TableRowColumn } from './table.types'
import { TableRow } from './table-row'

type Item = { id: string; name: string; count: number; tags: string[] }

const ITEMS: Item[] = [
    { id: '1', name: 'public.users', count: 14200, tags: ['kafka', 'http'] },
    { id: '2', name: 'public.orders', count: 38400, tags: ['kafka'] },
    { id: '3', name: 'public.products', count: 5100, tags: ['http', 'webhook'] },
]

const COLUMNS: TableRowColumn<Item>[] = [
    {
        key: 'name',
        header: 'Name',
        className: 'flex-1 truncate',
        cell: (item) => <span className="font-mono text-base text-ink">{item.name}</span>,
        mobile: {
            zone: 'title',
            render: (item) => <span className="min-w-0 truncate font-mono text-base text-ink">{item.name}</span>,
        },
    },
    {
        key: 'count',
        header: 'Count',
        className: 'w-32 text-right',
        cell: (item) => <span className="font-mono text-ink-muted text-sm">{item.count.toLocaleString('en-US')}</span>,
        mobile: {
            zone: 'meta',
            render: (item) => <span>{item.count.toLocaleString('en-US')} rows</span>,
        },
    },
    {
        key: 'tags',
        header: 'Tags',
        className: 'w-48',
        cell: (item) => (
            <div className="flex flex-wrap gap-1.5">
                {item.tags.map((tag) => (
                    <Badge key={tag} variant="neutral" className="font-mono">
                        {tag}
                    </Badge>
                ))}
            </div>
        ),
        mobile: {
            zone: 'badges',
            render: (item) => (
                <>
                    {item.tags.map((tag) => (
                        <Badge key={tag} variant="neutral" className="font-mono">
                            {tag}
                        </Badge>
                    ))}
                </>
            ),
        },
    },
]

type StoryArgs = {
    variant: 'default' | 'bordered'
    selectable: boolean
}

const meta: Meta<StoryArgs> = {
    title: 'UI Kit/Table',
    argTypes: {
        variant: {
            name: 'Variant',
            control: 'inline-radio',
            options: ['default', 'bordered'],
            table: { category: 'Style' },
        },
        selectable: { name: 'Selectable rows', control: 'boolean', table: { category: 'State' } },
    },
    args: { variant: 'default', selectable: false },
    render: ({ variant, selectable }) => {
        const [checked, setChecked] = useState<Set<string>>(new Set())
        const toggle = (id: string) =>
            setChecked((prev) => {
                const next = new Set(prev)
                next.has(id) ? next.delete(id) : next.add(id)
                return next
            })

        return (
            <div className="p-6">
                <Table
                    columns={COLUMNS}
                    variant={variant}
                    selectable={selectable}
                    header={<span className="font-semibold text-ink text-md">Items</span>}
                    headerAside={<span className="text-ink-subtle text-sm">{ITEMS.length} total</span>}
                >
                    {ITEMS.map((item) => (
                        <TableRow
                            key={item.id}
                            row={item}
                            columns={COLUMNS}
                            selectable={
                                selectable
                                    ? {
                                          id: `story-item-${item.id}`,
                                          checked: checked.has(item.id),
                                          onToggle: () => toggle(item.id),
                                      }
                                    : undefined
                            }
                        />
                    ))}
                </Table>
            </div>
        )
    },
}

export default meta

export const Default: StoryObj<StoryArgs> = {
    name: 'Table',
}

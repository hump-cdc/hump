import type { ReactNode } from 'react'
import { Fragment } from 'react'

import { cva } from 'class-variance-authority'

import { Checkbox } from '@/shared/ui/checkbox'
import { cn } from '@/shared/utils'

import type { MobileZone, TableRowColumn } from './table.types'
import { useTableContext } from './table-context'

const tableRowVariants = cva('border-hairline border-b py-3 last:border-0', {
    variants: {
        variant: {
            default: '',
            bordered: 'px-4 last:rounded-b-xl',
        },
    },
    defaultVariants: {
        variant: 'default',
    },
})

export type SelectableProps = {
    id: string
    checked: boolean
    onToggle: () => void
}

type TableRowProps<T> = {
    row: T
    columns: TableRowColumn<T>[]
    selectable?: SelectableProps
    mobileZoneClasses?: Partial<Record<MobileZone, string>>
}

const ZONE_ORDER: MobileZone[] = ['title', 'meta', 'fill', 'badges']

const ZONE_BASE: Record<MobileZone, string> = {
    title: 'flex items-center gap-2',
    meta: 'flex flex-wrap gap-x-4 gap-y-1 font-mono text-sm text-ink-subtle',
    fill: '',
    badges: 'flex flex-wrap gap-1.5',
}

export function TableRow<T>({ row, columns, selectable, mobileZoneClasses }: TableRowProps<T>): ReactNode {
    const { variant } = useTableContext()

    if (selectable) {
        const { id, checked, onToggle } = selectable
        return (
            <label
                htmlFor={id}
                className={cn(
                    tableRowVariants({ variant }),
                    'flex cursor-pointer items-start gap-3 transition-colors hover:bg-surface-2 lg:items-center',
                    checked && 'bg-primary-soft/40',
                )}
            >
                <Checkbox id={id} checked={checked} onCheckedChange={onToggle} className="mt-0.5 shrink-0 lg:mt-0" />
                <div className="flex min-w-0 flex-1 flex-col gap-2 lg:flex-row lg:items-center lg:gap-4">
                    {columns.map((col) => (
                        <Fragment key={col.key}>{col.cell(row)}</Fragment>
                    ))}
                </div>
            </label>
        )
    }

    return (
        <div className={tableRowVariants({ variant })}>
            <div className="hidden items-center gap-4 lg:flex">
                {columns.map((col) => (
                    <div key={col.key} className={col.className}>
                        {col.cell(row)}
                    </div>
                ))}
            </div>
            <div className="flex flex-col gap-2 lg:hidden">
                {ZONE_ORDER.map((zone) => {
                    const cols = columns.filter((col) => col.mobile?.zone === zone)
                    if (cols.length === 0) return null
                    return (
                        <div key={zone} className={cn(ZONE_BASE[zone], mobileZoneClasses?.[zone])}>
                            {cols.map(({ key, mobile }) =>
                                mobile?.className ? (
                                    <div key={key} className={mobile.className}>
                                        {mobile.render(row)}
                                    </div>
                                ) : (
                                    <Fragment key={key}>{mobile?.render(row)}</Fragment>
                                ),
                            )}
                        </div>
                    )
                })}
            </div>
        </div>
    )
}

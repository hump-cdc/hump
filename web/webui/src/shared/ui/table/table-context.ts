import { createContext, useContext } from 'react'

/** Visual variant of a table: `default` has no frame, `bordered` sits in a card. */
export type TableVariant = 'default' | 'bordered'

type TableContextValue = {
    variant: TableVariant
}

export const TableContext = createContext<TableContextValue>({ variant: 'default' })

/** Variant of the enclosing table, so its header and rows pad themselves to match. */
export const useTableContext = (): TableContextValue => useContext(TableContext)

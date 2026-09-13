import { type ReactNode, useCallback, useMemo } from 'react'

import type { RowHeightParams } from 'ag-grid-community'
import type { CustomCellRendererProps } from 'ag-grid-react'

import type { ExpandContext, InternalRow } from './data-table.helpers.expandable'
import type { DetailComponent } from './data-table.types'

/** Memoizes the expand context wrapping the row toggle. */
export const useExpandContext = (toggle: (id: string) => void): ExpandContext =>
    useMemo<ExpandContext>(() => ({ toggle }), [toggle])

/** Builds the full-width renderer that draws a detail row's content. */
export const useFullWidthCellRenderer = <T,>(getDetail?: DetailComponent<T>) =>
    useMemo(() => {
        const Detail = getDetail
        return ({ data }: CustomCellRendererProps<InternalRow<T>>): ReactNode => {
            if (!Detail || !data || data.kind !== 'detail') {
                return null
            }
            return (
                <div className="bg-surface-2 pt-0.5 pr-4 pb-4 pl-11">
                    <div className="overflow-hidden rounded-[10px] border border-hairline bg-canvas">
                        <Detail row={data.row} />
                    </div>
                </div>
            )
        }
    }, [getDetail])

/** Resolves a row's height, honoring per-detail-row heights. */
export const useRowHeight = <T,>(detailHeight: number | ((row: T) => number), rowHeight: number) =>
    useCallback(
        (params: RowHeightParams<InternalRow<T>>) => {
            const node = params.data
            if (node?.kind === 'detail') {
                return typeof detailHeight === 'function' ? detailHeight(node.row) : detailHeight
            }
            return rowHeight
        },
        [detailHeight, rowHeight],
    )

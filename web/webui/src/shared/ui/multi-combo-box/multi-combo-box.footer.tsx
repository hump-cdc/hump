import type { FC } from 'react'

type MultiComboBoxFooterProps = {
    selectedCount: number
    totalCount: number
    onSelectAll: () => void
}

export const MultiComboBoxFooter: FC<MultiComboBoxFooterProps> = ({ selectedCount, totalCount, onSelectAll }) => (
    <div className="flex items-center justify-between gap-2 border-hairline border-t px-3.5 py-2.5">
        <button
            type="button"
            onClick={onSelectAll}
            disabled={selectedCount === totalCount}
            className="rounded font-medium text-primary text-sm outline-none transition-colors hover:text-primary-hover focus-visible:ring-2 focus-visible:ring-primary/40 disabled:pointer-events-none disabled:opacity-50"
        >
            Select all
        </button>
        <span className="text-ink-tertiary text-sm">{selectedCount} selected</span>
    </div>
)

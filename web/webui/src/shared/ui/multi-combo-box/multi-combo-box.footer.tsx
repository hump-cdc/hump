import type { FC } from 'react'

type MultiComboBoxFooterProps = {
    selectedCount: number
}

export const MultiComboBoxFooter: FC<MultiComboBoxFooterProps> = ({ selectedCount }) => (
    <div className="flex items-center justify-end gap-2 border-hairline border-t px-3.5 py-2.5">
        <span className="text-ink-tertiary text-sm">{selectedCount} selected</span>
    </div>
)

import type { FC } from 'react'

type MultiComboBoxFooterProps = {
    selectedCount: number
}

export const MultiComboBoxFooter: FC<MultiComboBoxFooterProps> = ({ selectedCount }) => (
    <div className="flex items-center justify-end gap-2 border-t border-hairline px-3.5 py-2.5">
        <span className="text-sm text-ink-tertiary">{selectedCount} selected</span>
    </div>
)

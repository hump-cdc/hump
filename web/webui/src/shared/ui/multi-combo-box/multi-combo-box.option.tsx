import type { FC } from 'react'

import { Check } from 'lucide-react'

import { ComboboxItem } from '@/shared/ui/combobox'

import type { MultiComboBoxItem } from './multi-combo-box.types'

type MultiComboBoxOptionProps = {
    item: MultiComboBoxItem
}

export const MultiComboBoxOption: FC<MultiComboBoxOptionProps> = ({ item }) => (
    <ComboboxItem value={item}>
        <span className="flex size-4.5 shrink-0 items-center justify-center rounded-[5px] border-[1.5px] border-hairline-strong text-on-primary transition-colors group-data-selected/combobox-item:border-transparent group-data-selected/combobox-item:bg-primary">
            <Check className="size-3 opacity-0 group-data-selected/combobox-item:opacity-100" />
        </span>
        <span className="flex-1 truncate font-medium text-ink text-md">{item.label}</span>
        {item.aside !== undefined && (
            <span className="shrink-0 font-mono text-ink-tertiary text-sm tabular-nums">{item.aside}</span>
        )}
    </ComboboxItem>
)

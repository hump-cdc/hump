import type { FC, ReactNode, Ref } from 'react'

import { ChevronsUpDown } from 'lucide-react'

import {
    Combobox,
    ComboboxChip,
    ComboboxChips,
    ComboboxChipsInput,
    ComboboxClear,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxList,
    ComboboxValue,
} from '@/shared/ui/combobox'

import { MultiComboBoxFooter } from './multi-combo-box.footer'
import { useChipOverflow, useChipsRef, useMultiComboBoxSelection } from './multi-combo-box.helpers'
import { MultiComboBoxOption } from './multi-combo-box.option'
import type { MultiComboBoxItem } from './multi-combo-box.types'

type MultiComboBoxProps = {
    items: MultiComboBoxItem[]
    value?: string[]
    onValueChange?: (value: string[]) => void
    placeholder?: string
    emptyText?: string
    icon?: ReactNode
    invalid?: boolean
    disabled?: boolean
    className?: string
    ref?: Ref<HTMLDivElement>
}

export const MultiComboBox: FC<MultiComboBoxProps> = ({
    items,
    value,
    onValueChange,
    placeholder,
    emptyText = 'No items found.',
    icon,
    invalid,
    disabled,
    className,
    ref,
}) => {
    const { selectedItems, commit } = useMultiComboBoxSelection(items, value, onValueChange)
    const { anchor, setChipsRef } = useChipsRef(ref)
    const { containerRef, measureRef, visibleCount } = useChipOverflow(selectedItems.length)

    return (
        <Combobox
            multiple
            autoHighlight
            items={items}
            value={selectedItems}
            onValueChange={commit}
            itemToStringLabel={(item: MultiComboBoxItem) => item.label}
            isItemEqualToValue={(a: MultiComboBoxItem, b: MultiComboBoxItem) => a.value === b.value}
            disabled={disabled}
        >
            <ComboboxChips ref={setChipsRef} aria-invalid={invalid || undefined} className={className}>
                {icon && <span className="flex shrink-0 items-center text-ink-subtle [&_svg]:size-4">{icon}</span>}

                <div ref={containerRef} className="relative flex min-w-0 flex-1 items-center gap-1.5 overflow-hidden">
                    {/* Invisible measurement row: all chips at natural width, read by useChipOverflow. */}
                    <div
                        ref={measureRef}
                        aria-hidden
                        className="pointer-events-none invisible absolute -z-10 flex items-center gap-1.5"
                    >
                        {selectedItems.map((item) => (
                            <span
                                key={item.value}
                                className="inline-flex h-5.5 shrink-0 items-center gap-1 whitespace-nowrap rounded-md pr-1 pl-2 font-medium text-sm"
                            >
                                {item.label}
                                <span className="size-4" />
                            </span>
                        ))}
                    </div>

                    <ComboboxValue>
                        {(values: MultiComboBoxItem[]) => {
                            const allSelected = items.length > 0 && values.length === items.length
                            const visible = values.slice(0, visibleCount)
                            const hidden = values.length - visible.length

                            return (
                                <>
                                    {allSelected ? (
                                        <span className="flex h-5.5 shrink-0 select-none items-center rounded-md bg-surface-3 px-2 font-medium text-ink-muted text-sm">
                                            All
                                        </span>
                                    ) : (
                                        <>
                                            {visible.map((item) => (
                                                <ComboboxChip key={item.value}>{item.label}</ComboboxChip>
                                            ))}
                                            {hidden > 0 && (
                                                <span className="flex h-5.5 shrink-0 select-none items-center rounded-md bg-primary-soft px-2 font-semibold text-primary text-sm">
                                                    +{hidden}
                                                </span>
                                            )}
                                        </>
                                    )}
                                    <ComboboxChipsInput placeholder={values.length === 0 ? placeholder : undefined} />
                                </>
                            )
                        }}
                    </ComboboxValue>
                </div>

                {selectedItems.length > 0 && <ComboboxClear disabled={disabled} />}
                <ChevronsUpDown className="pointer-events-none size-4 shrink-0 text-ink-subtle" />
            </ComboboxChips>

            <ComboboxContent anchor={anchor}>
                <ComboboxList>
                    {(item: MultiComboBoxItem) => <MultiComboBoxOption key={item.value} item={item} />}
                </ComboboxList>
                <ComboboxEmpty>{emptyText}</ComboboxEmpty>
                <MultiComboBoxFooter
                    selectedCount={selectedItems.length}
                    totalCount={items.length}
                    onSelectAll={() => commit(items)}
                />
            </ComboboxContent>
        </Combobox>
    )
}

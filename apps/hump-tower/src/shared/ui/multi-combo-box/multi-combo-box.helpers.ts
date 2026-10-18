import type { Ref } from 'react'
import { useCallback, useLayoutEffect, useMemo, useRef, useState } from 'react'

import { useComboboxAnchor } from '@/shared/ui/combobox'

import type { MultiComboBoxItem } from './multi-combo-box.types'

const GAP = 6 // gap-1.5
const INPUT_MIN = 20 // minimal caret room kept for the search input
const OVERFLOW_RESERVE = 40 // room kept for the "+N" badge

/**
 * Computes how many chips fit on the single trigger line. The consumer renders an
 * invisible measurement row (all chips at natural width) referenced by `measureRef`
 * inside the `containerRef` element; the hook reads those widths and the available
 * width, then returns the number of chips to show before collapsing the rest into "+N".
 */
export const useChipOverflow = (count: number) => {
    const containerRef = useRef<HTMLDivElement>(null)
    const measureRef = useRef<HTMLDivElement>(null)
    const [visibleCount, setVisibleCount] = useState(count)

    const recompute = useCallback(() => {
        const container = containerRef.current
        const measure = measureRef.current
        if (!container || !measure) {
            return
        }

        const chips = Array.from(measure.children) as HTMLElement[]
        const total = chips.length
        if (total === 0) {
            setVisibleCount(0)
            return
        }

        const budget = container.clientWidth - INPUT_MIN
        let used = 0
        let visible = 0
        for (let i = 0; i < total; i++) {
            const width = chips[i].offsetWidth + (visible > 0 ? GAP : 0)
            const reserveForOverflow = i < total - 1 ? OVERFLOW_RESERVE + GAP : 0
            if (used + width + reserveForOverflow <= budget) {
                used += width
                visible++
            } else {
                break
            }
        }
        setVisibleCount(visible)
    }, [])

    // Recompute after every render so selection and label changes are picked up.
    // The measurement row always renders all chips, so widths are stable and this converges.
    useLayoutEffect(() => {
        recompute()
    })

    // Recompute when the trigger is resized.
    useLayoutEffect(() => {
        const container = containerRef.current
        if (!container) {
            return
        }
        const observer = new ResizeObserver(recompute)
        observer.observe(container)
        return () => observer.disconnect()
    }, [recompute])

    return { containerRef, measureRef, visibleCount }
}

/**
 * Manages the combobox selection for both controlled (`value`) and uncontrolled usage,
 * mapping the public `string[]` API to the item objects the combobox works with.
 */
export const useMultiComboBoxSelection = (
    items: MultiComboBoxItem[],
    value: string[] | undefined,
    onValueChange?: (value: string[]) => void,
) => {
    const [internalValue, setInternalValue] = useState<string[]>([])
    const selectedValues = value ?? internalValue

    const selectedItems = useMemo(() => {
        const byValue = new Map(items.map((item) => [item.value, item]))
        return selectedValues.map((v) => byValue.get(v)).filter((item): item is MultiComboBoxItem => item !== undefined)
    }, [items, selectedValues])

    const commit = useCallback(
        (next: MultiComboBoxItem[]) => {
            const nextValues = next.map((item) => item.value)
            if (value === undefined) {
                setInternalValue(nextValues)
            }
            onValueChange?.(nextValues)
        },
        [value, onValueChange],
    )

    return { selectedItems, commit }
}

/**
 * Creates the combobox anchor and a stable ref callback that points both the anchor
 * (for popover positioning) and an optional forwarded ref at the trigger element.
 */
export const useChipsRef = (ref?: Ref<HTMLDivElement>) => {
    const anchor = useComboboxAnchor()

    const setChipsRef = useCallback(
        (node: HTMLDivElement | null) => {
            anchor.current = node
            if (typeof ref === 'function') {
                ref(node)
            } else if (ref) {
                ref.current = node
            }
        },
        [anchor, ref],
    )

    return { anchor, setChipsRef }
}

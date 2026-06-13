import { useRef } from 'react'

import { Combobox as ComboboxPrimitive } from '@base-ui/react/combobox'

const useComboboxAnchor = () => useRef<HTMLDivElement | null>(null)

const useComboboxFilteredItems = ComboboxPrimitive.useFilteredItems

export { useComboboxAnchor, useComboboxFilteredItems }

import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import {
    Combobox,
    ComboboxChip,
    ComboboxChips,
    ComboboxChipsInput,
    ComboboxClear,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxItem,
    ComboboxList,
    ComboboxValue,
    useComboboxAnchor,
    useComboboxFilteredItems,
} from './index'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

const items = ['kafka', 'nats', 'http']

/** Lists the filtered items as rendered by the combobox's own hook. */
const FilteredItems = () => {
    const filtered = useComboboxFilteredItems<string>()

    return <output>{filtered.join(',')}</output>
}

/** An open multi-select combobox with every part of the kit in place. */
const Picker = () => {
    const anchor = useComboboxAnchor()

    return (
        <Combobox items={items} multiple defaultValue={['kafka', 'nats']} open>
            <ComboboxChips ref={anchor}>
                <ComboboxValue>
                    {(values: string[]) =>
                        values.map((value, index) => (
                            <ComboboxChip key={value} showRemove={index === 0}>
                                {value}
                            </ComboboxChip>
                        ))
                    }
                </ComboboxValue>
                <ComboboxChipsInput placeholder="Pick sinks" />
                <ComboboxClear />
            </ComboboxChips>
            <ComboboxContent anchor={anchor}>
                <ComboboxList>
                    {(item: string) => (
                        <ComboboxItem key={item} value={item}>
                            {item}
                        </ComboboxItem>
                    )}
                </ComboboxList>
                <ComboboxEmpty>Nothing found</ComboboxEmpty>
                <FilteredItems />
            </ComboboxContent>
        </Combobox>
    )
}

describe('Combobox', () => {
    test('renders chips, the input, clear button and the open list', async () => {
        const { findByRole, getAllByLabelText, getByLabelText, getByPlaceholderText } = render(<Picker />)
        await findByRole('listbox', { hidden: true })
        const slots = [...document.querySelectorAll('[data-slot]')].map((element) => element.getAttribute('data-slot'))

        expect(getByPlaceholderText('Pick sinks')).toBeDefined()
        expect(getByLabelText('Clear selection')).toBeDefined()
        expect(getAllByLabelText('Remove')).toHaveLength(1)
        expect(slots.filter((slot) => slot === 'combobox-chip')).toHaveLength(2)
        expect(slots.filter((slot) => slot === 'combobox-item')).toHaveLength(3)
        expect(slots).toEqual(
            expect.arrayContaining(['combobox-chips', 'combobox-list', 'combobox-content', 'combobox-empty']),
        )
        expect(document.querySelector('output')?.textContent).toBe('kafka,nats,http')
    })
})

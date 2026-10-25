import { createRef, useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render, waitFor } from '@testing-library/react'

import { TooltipProvider } from '@/shared/ui/tooltip'

import { MultiComboBox, type MultiComboBoxItem } from './index'
// Type-only module: `export type *` erases it, so load it here for the coverage report to list it.
import './multi-combo-box.types'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))

const items: MultiComboBoxItem[] = [
    { value: 'kafka', label: 'Kafka', aside: '3' },
    { value: 'nats', label: 'NATS' },
    { value: 'http', label: 'HTTP' },
    { value: 'rabbitmq', label: 'RabbitMQ' },
]

/** Overrides an element size getter on every `HTMLElement` until the returned function restores it. */
const stubSize = (property: 'clientWidth' | 'offsetWidth', size: number): (() => void) => {
    const original = Object.getOwnPropertyDescriptor(HTMLElement.prototype, property)
    Object.defineProperty(HTMLElement.prototype, property, { configurable: true, get: () => size })

    return () => {
        if (original) Object.defineProperty(HTMLElement.prototype, property, original)
    }
}

/** A `ResizeObserver` that never fires on its own but keeps every callback for the test to call. */
class RecordingResizeObserver implements ResizeObserver {
    static callbacks: ResizeObserverCallback[] = []

    constructor(callback: ResizeObserverCallback) {
        RecordingResizeObserver.callbacks.push(callback)
    }

    observe(): void {}

    unobserve(): void {}

    disconnect(): void {}
}

/** Opens the popup from the chips input and waits for the list. */
const openList = async (input: HTMLElement): Promise<void> => {
    await act(async () => {
        input.focus()
        fireEvent.focusIn(input)
        fireEvent.keyDown(input, { key: 'ArrowDown' })
    })
    await waitFor(() => expect(document.body.querySelector('[role="listbox"]')).not.toBeNull())
}

/** Chip labels currently shown on the trigger, measurement row excluded. */
const chipLabels = (): string[] =>
    [...document.body.querySelectorAll('[data-slot="combobox-chip"]')].map((chip) => chip.textContent ?? '')

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
    // React DOM loaded without a DOM (bun shares modules across test files) watches the focused text
    // input through IE's `attachEvent`; happy-dom has none, so give its elements a no-op one.
    Object.assign(HTMLElement.prototype, { attachEvent: () => {}, detachEvent: () => {} })
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('MultiComboBox', () => {
    test('keeps its own selection while uncontrolled', async () => {
        const ref = createRef<HTMLDivElement>()
        const view = render(<MultiComboBox items={items} placeholder="Pick sinks" emptyText="Nothing" ref={ref} />)
        const input = view.getByPlaceholderText('Pick sinks')

        expect(ref.current?.dataset.slot).toBe('combobox-chips')
        expect(view.queryByLabelText('Clear selection')).toBeNull()

        await openList(input)
        expect(view.getByText('3').className).toContain('tabular-nums')
        expect(view.getByText('0 selected')).toBeDefined()

        await act(async () => fireEvent.click(view.getByRole('option', { name: /NATS/ })))
        expect(view.getByText('1 selected')).toBeDefined()

        await act(async () => fireEvent.click(view.getByText('Select all')))
        expect(view.getByText('4 selected')).toBeDefined()
        expect(view.getByText('All')).toBeDefined()
        expect((view.getByText('Select all') as HTMLButtonElement).disabled).toBe(true)

        await act(async () => fireEvent.click(view.getByLabelText('Clear selection')))
        expect(view.getByText('0 selected')).toBeDefined()

        await act(async () => {
            fireEvent.input(input, { target: { value: 'zzz' } })
            fireEvent.keyDown(input, { key: 'z' })
        })
        expect(view.getByText('Nothing')).toBeDefined()
    })

    test('reports changes while controlled and shows the icon hint', async () => {
        const restoreClient = stubSize('clientWidth', 1000)
        const restoreOffset = stubSize('offsetWidth', 50)
        const onValueChange = mock((_value: string[]) => {})
        const ref = mock((_node: HTMLDivElement | null) => {})
        const view = render(
            <TooltipProvider>
                <MultiComboBox
                    items={items}
                    value={['kafka']}
                    onValueChange={onValueChange}
                    icon={<svg data-testid="icon" />}
                    hint="Filter by sink"
                    invalid
                    ref={ref}
                />
            </TooltipProvider>,
        )

        expect(view.getByTestId('icon').parentElement?.dataset.slot).toBe('tooltip-trigger')
        expect(ref).toHaveBeenCalled()
        expect(view.container.querySelector('[data-slot="combobox-chips"]')?.getAttribute('aria-invalid')).toBe('true')
        expect(chipLabels()).toEqual(['Kafka'])

        await act(async () => fireEvent.click(view.getByLabelText('Clear selection')))
        expect(onValueChange).toHaveBeenLastCalledWith([])
        expect(chipLabels()).toEqual(['Kafka'])
        restoreClient()
        restoreOffset()
    })

    test('resets to every item with `clearToAll`', async () => {
        const onValueChange = mock((_value: string[]) => {})
        const view = render(
            <MultiComboBox
                items={items}
                value={['kafka', 'nats']}
                onValueChange={onValueChange}
                icon={<svg data-testid="icon" />}
                clearToAll
            />,
        )

        expect(view.getByTestId('icon').parentElement?.dataset.slot).toBeUndefined()

        await act(async () => fireEvent.click(view.getByLabelText('Clear selection')))
        expect(onValueChange).toHaveBeenLastCalledWith(['kafka', 'nats', 'http', 'rabbitmq'])
    })

    test('collapses the chips that do not fit into a counter', () => {
        const restoreClient = stubSize('clientWidth', 200)
        const restoreOffset = stubSize('offsetWidth', 60)
        try {
            render(<MultiComboBox items={items} value={['kafka', 'nats', 'http']} />)

            expect(chipLabels()).toEqual(['Kafka', 'NATS'])
            expect(document.body.textContent).toContain('+1')
        } finally {
            restoreClient()
            restoreOffset()
        }
    })

    test('ignores a resize that lands after unmount', () => {
        const original = globalThis.ResizeObserver
        globalThis.ResizeObserver = RecordingResizeObserver
        RecordingResizeObserver.callbacks = []
        try {
            const view = render(<MultiComboBox items={items} value={['kafka']} />)
            view.unmount()

            const observer = new RecordingResizeObserver(() => {})
            for (const callback of RecordingResizeObserver.callbacks) {
                expect(() => callback([], observer)).not.toThrow()
            }
        } finally {
            globalThis.ResizeObserver = original
        }
    })
})

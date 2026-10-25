import { createRef } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, fireEvent, render } from '@testing-library/react'

import { TextInput } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

/**
 * Types a value the way React DOM listens for it. Loaded without a DOM (bun shares modules across
 * test files), React DOM falls back to watching keys on the focused input instead of `input` events.
 */
const typeInto = (input: HTMLInputElement, value: string): void => {
    Object.assign(input, { attachEvent: () => {}, detachEvent: () => {} })
    fireEvent.focusIn(input)
    fireEvent.input(input, { target: { value } })
    fireEvent.keyDown(input)
}

describe('TextInput', () => {
    test('keeps its own value while uncontrolled and clears it', () => {
        const onValueChange = mock((_value: string) => {})
        const ref = createRef<HTMLInputElement>()
        const view = render(
            <TextInput
                defaultValue="abc"
                onValueChange={onValueChange}
                ref={ref}
                icon={<svg data-testid="icon" />}
                suffix={<span>ms</span>}
                font="sans"
                invalid
            />,
        )
        const input = view.getByRole('textbox') as HTMLInputElement

        expect(ref.current).toBe(input)
        expect(input.className).toContain('font-sans')
        expect(view.getByTestId('icon')).toBeDefined()
        expect(view.getByText('ms')).toBeDefined()
        expect(view.container.firstElementChild?.getAttribute('aria-invalid')).toBe('true')

        typeInto(input, 'abcd')
        expect(input.value).toBe('abcd')
        expect(onValueChange).toHaveBeenLastCalledWith('abcd')

        fireEvent.click(view.getByLabelText('Clear'))
        expect(input.value).toBe('')
        expect(onValueChange).toHaveBeenLastCalledWith('')
        expect(document.activeElement).toBe(input)
        expect(view.queryByLabelText('Clear')).toBeNull()
    })

    test('reports changes without applying them while controlled', () => {
        const onValueChange = mock((_value: string) => {})
        const ref = mock((_node: HTMLInputElement | null) => {})
        const view = render(<TextInput value="fixed" onValueChange={onValueChange} ref={ref} type="password" />)
        const input = view.container.querySelector('input') as HTMLInputElement

        expect(ref).toHaveBeenCalledWith(input)
        expect(input.className).toContain('font-mono')

        typeInto(input, 'other')
        expect(onValueChange).toHaveBeenCalledWith('other')
        expect(input.value).toBe('fixed')
    })

    test('hides the clear button while disabled', () => {
        const view = render(<TextInput defaultValue="abc" disabled />)

        expect(view.queryByLabelText('Clear')).toBeNull()
    })
})

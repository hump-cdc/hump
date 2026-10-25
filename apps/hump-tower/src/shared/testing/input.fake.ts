import { act, fireEvent, waitFor } from '@testing-library/react'

/**
 * Types `value` into a text input. `change` reaches `onChange` when React DOM listens for `input`
 * events; the `keyup` after it does when React DOM fell back to the `focusin` tracking.
 */
export const typeInto = (input: HTMLElement, value: string): void => {
    focusInput(input)
    fireEvent.change(input, { target: { value } })
    fireEvent.keyUp(input)
}

/** Opens a `MultiComboBox` popup from its input and waits for the option list. */
export const openComboBox = async (input: HTMLElement): Promise<void> => {
    await act(async () => {
        focusInput(input)
        fireEvent.click(input)
        fireEvent.keyDown(input, { key: 'ArrowDown' })
    })
    await waitFor(() => {
        if (!document.body.querySelector('[role="listbox"]')) throw new Error('The option list is not open')
    })
}

/**
 * Focuses a text input the way a user does before typing into it. React DOM loaded before the DOM
 * existed falls back to tracking text inputs through `focusin`, and fails on a key press without one.
 */
const focusInput = (input: HTMLElement): void => {
    fireEvent.focusIn(input)
}

import { type ComponentProps, createRef, type FC, useLayoutEffect, useState } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'
import type { PrismEditor } from 'prism-react-editor'

import { TooltipProvider } from '@/shared/ui/tooltip'

import { type WhereField, WhereInput } from './index'
// Type-only module: `import type` erases it, so load it here for the coverage report to list it.
import './where-input.types'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))
// prism-react-editor reads `document` once, when first loaded, which in a shared test process may be
// before any DOM exists. Its caret tracker and `insertText` (`document.execCommand`) then break, so
// replace those two with DOM-independent stand-ins; the editor itself copes.
mock.module('prism-react-editor/cursor', () => ({ useCursorPosition: () => {} }))
mock.module('prism-react-editor/utils', () => ({ insertText: fakeInsertText }))

const fields: WhereField[] = [{ name: 'status', type: 'text' }, { name: 'state' }, { name: 'id', type: 'bigint' }]

type HarnessProps = Omit<ComponentProps<typeof WhereInput>, 'value' | 'onValueChange'> & {
    initialValue?: string
    onValueChange?: (value: string) => void
}

/** `WhereInput` over its own state, so clearing and restoring flow back into the editor. */
const Harness: FC<HarnessProps> = ({ initialValue = '', onValueChange, ...props }) => {
    const [value, setValue] = useState(initialValue)

    return (
        <TooltipProvider>
            <WhereInput
                value={value}
                onValueChange={(next) => {
                    onValueChange?.(next)
                    setValue(next)
                }}
                {...props}
            />
        </TooltipProvider>
    )
}

/** Renders the harness and returns the editor textarea alongside the queries. */
const renderInput = (props: HarnessProps = {}) => {
    const view = render(<Harness fields={fields} {...props} />)
    const textarea = view.getByRole('combobox') as HTMLTextAreaElement

    return { view, textarea }
}

/** Replaces the editor text and puts the caret at its end, as typing would. */
const type = async (textarea: HTMLTextAreaElement, text: string): Promise<void> => {
    await act(async () => {
        textarea.value = text
        textarea.setSelectionRange(text.length, text.length)
        fireEvent.input(textarea)
    })
}

/** Presses a key in the editor. */
const press = async (textarea: HTMLTextAreaElement, init: KeyboardEventInit): Promise<void> => {
    await act(async () => {
        fireEvent.keyDown(textarea, init)
    })
}

/** Focuses the editor textarea. */
const focus = async (textarea: HTMLTextAreaElement): Promise<void> => {
    await act(async () => textarea.focus())
}

/** Labels of the open suggestion list, empty while it is closed. */
const optionLabels = (): string[] =>
    [...document.body.querySelectorAll('[role="option"]')].map((option) => option.textContent ?? '')

/** Index of the highlighted option. */
const activeIndex = (): number =>
    [...document.body.querySelectorAll('[role="option"]')].findIndex((option) =>
        option.hasAttribute('data-highlighted'),
    )

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

describe('WhereInput', () => {
    test('shows a highlighted one-line preview while collapsed', () => {
        const { view, textarea } = renderInput({ initialValue: "\nstatus = 'a'\nand id > 1", invalid: true })

        expect(textarea.value).toBe("\nstatus = 'a'\nand id > 1")
        expect(view.getByText('WHERE').className).toContain('text-danger')
        expect(view.container.querySelector('.hump-where-collapsed')).not.toBeNull()
        expect(view.getByText('…')).toBeDefined()
        expect(view.container.querySelector('[aria-invalid="true"]')).not.toBeNull()
    })

    test('expands while focused and collapses again on blur', async () => {
        const onBlur = mock(() => {})
        const { view, textarea } = renderInput({ initialValue: 'id > 1', onBlur })
        const panel = view.container.querySelector<HTMLElement>('[aria-invalid], .absolute') as HTMLElement

        await focus(textarea)
        expect(panel.style.height).toBe('76px')
        expect(view.container.querySelector('.hump-where-collapsed')).toBeNull()

        await act(async () => textarea.blur())
        expect(panel.style.height).toBe('36px')
        expect(onBlur).toHaveBeenCalledTimes(1)
    })

    test('reports typed text and clears it', async () => {
        const onValueChange = mock((_value: string) => {})
        const { view, textarea } = renderInput({ initialValue: 'id', onValueChange })

        await focus(textarea)
        await type(textarea, 'id > 1')
        expect(onValueChange).toHaveBeenLastCalledWith('id > 1')

        await act(async () => fireEvent.click(view.getByLabelText('Clear')))
        expect(onValueChange).toHaveBeenLastCalledWith('')
        expect(textarea.value).toBe('')
        expect(view.queryByLabelText('Clear')).toBeNull()
    })

    test('restores the suggested condition while empty', async () => {
        const onValueChange = mock((_value: string) => {})
        const { view, textarea } = renderInput({ suggestion: "status = 'active'", onValueChange })

        await focus(textarea)
        await act(async () => fireEvent.click(view.getByLabelText('Restore suggested filter')))

        expect(onValueChange).toHaveBeenLastCalledWith("status = 'active'")
        expect(textarea.value).toBe("status = 'active'")
    })

    test('suggests fields while typing and navigates them with the keyboard', async () => {
        const { textarea } = renderInput()

        await focus(textarea)
        await type(textarea, 'st')
        expect(optionLabels().slice(0, 2)).toEqual(['statustext', 'state'])
        expect(textarea.getAttribute('aria-expanded')).toBe('true')
        expect(textarea.getAttribute('aria-activedescendant')).toEndWith('-0')

        await press(textarea, { key: 'ArrowDown' })
        expect(activeIndex()).toBe(1)
        await press(textarea, { key: 'ArrowUp' })
        await press(textarea, { key: 'ArrowUp' })
        expect(activeIndex()).toBe(optionLabels().length - 1)

        await press(textarea, { key: 'Escape' })
        expect(optionLabels()).toEqual([])
        await press(textarea, { key: 'ArrowDown' })
        expect(optionLabels()).toEqual([])
    })

    test('inserts the active suggestion on Enter and on Tab', async () => {
        const onValueChange = mock((_value: string) => {})
        const { textarea } = renderInput({ onValueChange })

        await focus(textarea)
        await type(textarea, 'st')
        await press(textarea, { key: 'Enter' })
        expect(onValueChange).toHaveBeenLastCalledWith('status')
        expect(optionLabels()).toEqual([])

        await type(textarea, 'status = 1 and sta')
        await press(textarea, { key: 'ArrowDown' })
        await press(textarea, { key: 'Tab' })
        expect(onValueChange).toHaveBeenLastCalledWith('status = 1 and state')
    })

    test('highlights and accepts suggestions with the mouse', async () => {
        const onValueChange = mock((_value: string) => {})
        const { textarea } = renderInput({ onValueChange })

        await focus(textarea)
        await type(textarea, 'i')
        const options = [...document.body.querySelectorAll<HTMLElement>('[role="option"]')]
        expect(options.length).toBeGreaterThan(1)

        await act(async () => {
            fireEvent.mouseMove(options[1])
            fireEvent.mouseMove(options[1])
        })
        expect(activeIndex()).toBe(1)

        await act(async () => {
            fireEvent.mouseDown(options[1])
            fireEvent.click(options[1])
        })
        expect(onValueChange).toHaveBeenCalled()
        expect(optionLabels()).toEqual([])
    })

    test('opens suggestions on Ctrl+Space and closes them when nothing matches', async () => {
        const { textarea } = renderInput()

        await focus(textarea)
        await press(textarea, { key: ' ', code: 'Space', ctrlKey: true })
        await press(textarea, { key: ' ', code: 'Space', metaKey: true })
        expect(optionLabels().length).toBeGreaterThan(0)

        await type(textarea, 'zzzz')
        expect(optionLabels()).toEqual([])
    })

    test('closes suggestions when the selection is not a caret', async () => {
        const { textarea } = renderInput()

        await focus(textarea)
        await type(textarea, 'st')
        expect(optionLabels().length).toBeGreaterThan(0)

        await act(async () => {
            textarea.setSelectionRange(0, 2)
            fireEvent.input(textarea)
        })
        expect(optionLabels()).toEqual([])
    })

    test('closes suggestions when the popup is dismissed', async () => {
        const { textarea } = renderInput()

        await focus(textarea)
        await type(textarea, 'st')
        expect(optionLabels().length).toBeGreaterThan(0)

        await act(async () => {
            fireEvent.keyDown(document.body, { key: 'Escape' })
        })
        expect(textarea.getAttribute('aria-expanded')).toBe('false')
    })

    test('keeps focus in the editor when the panel chrome is pressed', async () => {
        const { view, textarea } = renderInput()

        const onTextarea = fireEvent.mouseDown(textarea)
        expect(onTextarea).toBe(true)

        await act(async () => {
            expect(fireEvent.mouseDown(view.getByText('WHERE'))).toBe(false)
        })
        expect(document.activeElement).toBe(textarea)
        expect(fireEvent.click(view.getByText('WHERE'))).toBe(false)
    })

    test('hands the textarea to function and object refs', () => {
        const objectRef = createRef<HTMLTextAreaElement>()
        const functionRef = mock((_node: HTMLTextAreaElement | null) => {})

        const first = renderInput({ ref: objectRef })
        expect(objectRef.current).toBe(first.textarea)
        cleanup()

        const second = renderInput({ ref: functionRef })
        expect(functionRef).toHaveBeenCalledWith(second.textarea)
    })

    test('hides the clear and restore buttons while disabled', async () => {
        const { view, textarea } = renderInput({ initialValue: 'id', disabled: true, suggestion: 'id > 1' })

        expect(textarea.disabled).toBe(true)
        await focus(textarea)
        expect(view.queryByLabelText('Clear')).toBeNull()
        expect(view.queryByLabelText('Restore suggested filter')).toBeNull()
    })
})

/**
 * Stand-in for prism's `insertText`: replaces `start`–`end` with `text`, moves the caret and fires
 * `input` as a real edit would. The original goes through `document.execCommand`, which happy-dom lacks.
 */
function fakeInsertText(
    editor: PrismEditor,
    text: string,
    start: number,
    end: number,
    caretStart: number,
    caretEnd: number,
): void {
    const { textarea } = editor
    if (!textarea) return
    textarea.value = textarea.value.slice(0, start) + text + textarea.value.slice(end)
    textarea.setSelectionRange(caretStart, caretEnd)
    textarea.dispatchEvent(new Event('input', { bubbles: true }))
}

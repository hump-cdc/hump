import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'

import { Toolbar } from './index'

// Radix picks its layout effect when first imported, as a no-op without a DOM; bun shares modules
// across test files, so point it at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))

/** The happy-dom window, whose `matchMedia` follows the viewport set here. */
type HappyDomWindow = { happyDOM: { setViewport: (viewport: { width: number }) => void } }

/** Whether the global window is happy-dom's, i.e. carries its `happyDOM` controls. */
const isHappyDomWindow = (value: object): value is HappyDomWindow => 'happyDOM' in value

/** Resizes the happy-dom viewport, which `useIsMobile` reads through `matchMedia`. */
const setViewportWidth = (width: number): void => {
    if (!isHappyDomWindow(window)) throw new Error('happy-dom is not registered')
    window.happyDOM.setViewport({ width })
}

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(() => {
    cleanup()
    setViewportWidth(1024)
})

describe('Toolbar', () => {
    test('lays the controls out in a row on desktop', () => {
        const view = render(
            <Toolbar actions={<button type="button">Reset</button>} className="x">
                <input aria-label="Search" />
            </Toolbar>,
        )

        expect(view.getByLabelText('Search').parentElement?.className).toContain('items-center')
        expect(view.getByText('Reset')).toBeDefined()
        expect(view.queryByText('Filters')).toBeNull()
        expect(view.container.querySelector('.x')).not.toBeNull()
    })

    test('moves the controls into a sheet on mobile', async () => {
        setViewportWidth(500)
        const view = render(
            <Toolbar label="Options" actions={<button type="button">Reset</button>}>
                <input aria-label="Search" />
            </Toolbar>,
        )

        expect(view.queryByLabelText('Search')).toBeNull()
        expect(view.getByText('Reset')).toBeDefined()

        await act(async () => fireEvent.click(view.getByRole('button', { name: 'Options' })))

        expect(view.getByLabelText('Search')).toBeDefined()
        expect(document.body.querySelector('[data-slot="sheet-title"]')?.textContent).toBe('Options')
    })
})

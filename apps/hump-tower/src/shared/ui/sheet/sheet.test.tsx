import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'

import { Sheet, SheetClose, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from './index'

// Radix picks its layout effect when first imported, as a no-op without a DOM; bun shares modules
// across test files, so point it at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

type SampleProps = { portal?: boolean; showCloseButton?: boolean; showOverlay?: boolean }

/** A sheet with every part in place, opened by its trigger. */
const Sample = ({ portal, showCloseButton, showOverlay }: SampleProps) => (
    <Sheet>
        <SheetTrigger>Open</SheetTrigger>
        <SheetContent side="left" portal={portal} showCloseButton={showCloseButton} showOverlay={showOverlay}>
            <SheetHeader>
                <SheetTitle>Filters</SheetTitle>
                <SheetDescription>Narrow the list</SheetDescription>
            </SheetHeader>
            <SheetClose>Done</SheetClose>
        </SheetContent>
    </Sheet>
)

describe('Sheet', () => {
    test('opens through a portal with its overlay and close button', async () => {
        const view = render(<Sample />)

        await act(async () => fireEvent.click(view.getByText('Open')))

        const content = document.body.querySelector('[data-slot="sheet-content"]')
        expect(content?.className).toContain('left-0')
        expect(view.container.contains(content)).toBe(false)
        expect(document.body.querySelector('[data-slot="sheet-overlay"]')).not.toBeNull()
        expect(document.body.querySelector('[data-slot="sheet-header"]')).not.toBeNull()
        expect(view.getByText('Filters').dataset.slot).toBe('sheet-title')
        expect(view.getByText('Narrow the list').dataset.slot).toBe('sheet-description')
        expect(view.getByText('Close').className).toBe('sr-only')

        await act(async () => fireEvent.click(view.getByText('Done')))
        expect(document.body.querySelector('[data-slot="sheet-content"]')).toBeNull()
    })

    test('renders inline without the overlay and close button when asked to', async () => {
        const view = render(<Sample portal={false} showCloseButton={false} showOverlay={false} />)

        await act(async () => fireEvent.click(view.getByText('Open')))

        expect(view.container.querySelector('[data-slot="sheet-content"]')).not.toBeNull()
        expect(document.body.querySelector('[data-slot="sheet-overlay"]')).toBeNull()
        expect(view.queryByText('Close')).toBeNull()
    })
})

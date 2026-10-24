import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from './index'

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

describe('Tooltip', () => {
    test('renders the trigger and, while open, the content in a portal', () => {
        const view = render(
            <TooltipProvider>
                <Tooltip open>
                    <TooltipTrigger>Hover me</TooltipTrigger>
                    <TooltipContent variant="primary" className="x">
                        Hint
                    </TooltipContent>
                </Tooltip>
            </TooltipProvider>,
        )

        expect(view.getByText('Hover me').dataset.slot).toBe('tooltip-trigger')
        const content = document.body.querySelector('[data-slot="tooltip-content"]')
        expect(content?.className).toContain('bg-primary')
        expect(content?.className).toContain('x')
        expect(content?.textContent).toContain('Hint')
    })

    test('renders no content while closed', () => {
        render(
            <TooltipProvider>
                <Tooltip>
                    <TooltipTrigger>Hover me</TooltipTrigger>
                    <TooltipContent>Hint</TooltipContent>
                </Tooltip>
            </TooltipProvider>,
        )

        expect(document.body.querySelector('[data-slot="tooltip-content"]')).toBeNull()
    })
})

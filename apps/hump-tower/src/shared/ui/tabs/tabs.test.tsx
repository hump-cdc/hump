import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'

import { Tabs, TabsContent, TabsList, TabsTrigger } from './index'

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

type SampleProps = { defaultValue?: string; variant?: 'pill' | 'underline' }

/** Two tabs with their panels; returns the indicator element too. */
const renderTabs = ({ defaultValue, variant }: SampleProps) => {
    const view = render(
        <Tabs defaultValue={defaultValue} className="x">
            <TabsList variant={variant}>
                <TabsTrigger value="one">One</TabsTrigger>
                <TabsTrigger value="two">Two</TabsTrigger>
            </TabsList>
            <TabsContent value="one">First panel</TabsContent>
            <TabsContent value="two">Second panel</TabsContent>
        </Tabs>,
    )
    const list = view.getByRole('tablist')
    const indicator = list.firstElementChild as HTMLElement

    return { view, list, indicator }
}

describe('Tabs', () => {
    test('shows the pill indicator over the active trigger and follows the selection', async () => {
        const { view, list, indicator } = renderTabs({ defaultValue: 'one' })

        expect(view.container.querySelector('[data-slot="tabs"]')?.className).toContain('x')
        expect(list.dataset.variant).toBe('pill')
        expect(indicator.style.opacity).toBe('1')
        expect(indicator.className).toContain('rounded-md')
        expect(view.getByText('First panel').dataset.slot).toBe('tabs-content')

        await act(async () => {
            fireEvent.mouseDown(view.getByText('Two'))
        })
        expect(view.getByText('Two').dataset.state).toBe('active')
        expect(view.getByText('Second panel')).toBeDefined()
    })

    test('draws the underline indicator along the bottom edge', () => {
        const { list, indicator } = renderTabs({ defaultValue: 'two', variant: 'underline' })

        expect(list.dataset.variant).toBe('underline')
        expect(indicator.className).toContain('bg-primary')
        expect(indicator.style.height).toBe('2px')
    })

    test('hides the indicator while no trigger is active', () => {
        const { indicator } = renderTabs({})

        expect(indicator.style.opacity).toBe('0')
    })
})

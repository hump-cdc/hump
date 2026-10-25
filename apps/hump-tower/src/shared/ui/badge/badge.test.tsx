import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { Badge } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('Badge', () => {
    test('renders the dot, icon, title and children in a span', () => {
        const { container } = render(
            <Badge dot dotAnimated icon={<svg />} title="Status" variant="healthy" border>
                OK
            </Badge>,
        )
        const badge = container.firstElementChild

        expect(badge?.tagName).toBe('SPAN')
        expect(badge?.querySelector('[data-slot=dot]')?.className).toContain('animate-ping')
        expect(badge?.querySelector('[data-slot=icon]')).not.toBeNull()
        expect(badge?.querySelector('[data-slot=title]')?.textContent).toBe('Status')
        expect(badge?.textContent).toBe('StatusOK')
    })

    test('renders only the children by default', () => {
        const { container } = render(<Badge>plain</Badge>)

        expect(container.querySelector('[data-slot]')).toBeNull()
        expect(container.textContent).toBe('plain')
    })

    test('merges onto the child element with asChild', () => {
        const { container } = render(
            <Badge asChild dot>
                <a href="/x">Open the docs</a>
            </Badge>,
        )
        const link = container.firstElementChild

        expect(link?.tagName).toBe('A')
        expect(link?.className).toContain('inline-flex')
        expect(link?.querySelector('[data-slot=dot]')).toBeNull()
    })
})

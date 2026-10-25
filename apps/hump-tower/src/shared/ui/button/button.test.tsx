import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { Button } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('Button', () => {
    test('shows a spinner and disables itself while loading', () => {
        const { getByRole } = render(<Button loading>Save</Button>)
        const button = getByRole('button')

        expect(button.hasAttribute('disabled')).toBe(true)
        expect(button.querySelector('.animate-spin')).not.toBeNull()
        expect(button.textContent).toBe('Save')
    })

    test('stays enabled with no spinner by default', () => {
        const { getByRole } = render(<Button variant="ghost">Save</Button>)
        const button = getByRole('button')

        expect(button.hasAttribute('disabled')).toBe(false)
        expect(button.querySelector('.animate-spin')).toBeNull()
    })

    test('merges onto the child element with asChild', () => {
        const { container } = render(
            <Button asChild loading>
                <a href="/x">Open the docs</a>
            </Button>,
        )
        const link = container.firstElementChild

        expect(link?.tagName).toBe('A')
        expect(link?.querySelector('.animate-spin')).toBeNull()
    })
})

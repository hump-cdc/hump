import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { NoPayloadPlaceholder, Placeholder } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('Placeholder', () => {
    test('renders the icon, title and description', () => {
        const view = render(
            <Placeholder
                icon={<svg data-testid="icon" />}
                title="Nothing here"
                description="Try again"
                className="x"
            />,
        )

        expect(view.getByTestId('icon')).toBeDefined()
        expect(view.getByText('Nothing here')).toBeDefined()
        expect(view.getByText('Try again')).toBeDefined()
        expect(view.container.firstElementChild?.className).toContain('x')
    })

    test('leaves out the icon badge and description when absent', () => {
        const view = render(<Placeholder title="Bare" />)

        expect(view.container.firstElementChild?.children).toHaveLength(1)
    })
})

describe('NoPayloadPlaceholder', () => {
    test('renders the fixed empty-payload copy', () => {
        const view = render(<NoPayloadPlaceholder data-testid="placeholder" />)

        expect(view.getByText('No payload')).toBeDefined()
        expect(view.getByText('No message body for the operation.')).toBeDefined()
        expect(view.getByTestId('placeholder').querySelector('svg')).not.toBeNull()
    })
})

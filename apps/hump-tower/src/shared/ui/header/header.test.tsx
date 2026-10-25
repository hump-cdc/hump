import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { Header, HeaderTitle } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('Header', () => {
    test('renders a header with its title and merges class names', () => {
        const view = render(
            <Header className="custom">
                <HeaderTitle className="title">Flows</HeaderTitle>
            </Header>,
        )

        const header = view.container.querySelector('[data-slot="header"]')
        expect(header?.tagName).toBe('HEADER')
        expect(header?.className).toContain('custom')
        expect(view.getByRole('heading', { level: 1 }).className).toContain('title')
        expect(view.getByText('Flows').dataset.slot).toBe('header-title')
    })
})

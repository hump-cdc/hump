import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { Mark } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('Mark', () => {
    test('renders a mark element with merged class names', () => {
        const view = render(<Mark className="custom">orders</Mark>)

        const mark = view.getByText('orders')
        expect(mark.tagName).toBe('MARK')
        expect(mark.dataset.slot).toBe('mark')
        expect(mark.className).toContain('custom')
        expect(mark.className).toContain('font-mono')
    })
})

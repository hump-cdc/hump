import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { Code } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('Code', () => {
    test('highlights the value with the given grammar', () => {
        const { container } = render(<Code value="SELECT 1" lang="sql" className="h-10" />)
        const wrapper = container.firstElementChild

        expect(wrapper?.className).toContain('h-10')
        expect(wrapper?.textContent).toContain('SELECT')
    })

    test('pretty-prints detected JSON when format is on', () => {
        const { container } = render(<Code value='{"a":1}' format lineNumbers={false} />)

        expect(container.textContent).toContain('"a": 1')
    })

    test('shows the value as written when format is off', () => {
        const { container } = render(<Code value='{"a":1}' />)

        expect(container.textContent).toContain('{"a":1}')
    })
})

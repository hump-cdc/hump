import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, fireEvent, render } from '@testing-library/react'

import { Checkbox } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('Checkbox', () => {
    test('marks itself invalid and reports a click', () => {
        const onCheckedChange = mock((_checked: boolean | 'indeterminate') => {})
        const { getByRole } = render(<Checkbox invalid onCheckedChange={onCheckedChange} />)
        const checkbox = getByRole('checkbox')

        expect(checkbox.getAttribute('aria-invalid')).toBe('true')
        fireEvent.click(checkbox)
        expect(onCheckedChange).toHaveBeenCalledWith(true)
    })

    test('omits aria-invalid and renders the indicator when checked', () => {
        const { getByRole } = render(<Checkbox checked />)
        const checkbox = getByRole('checkbox')

        expect(checkbox.hasAttribute('aria-invalid')).toBe(false)
        expect(checkbox.querySelector('svg')).not.toBeNull()
    })
})

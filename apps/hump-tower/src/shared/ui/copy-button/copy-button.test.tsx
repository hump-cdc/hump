import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, fireEvent, render, waitFor } from '@testing-library/react'

import { CopyButton } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('CopyButton', () => {
    test('copies the value and swaps to the check icon', async () => {
        const writeText = mock((_text: string) => Promise.resolve())
        Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
        const { getByRole } = render(<CopyButton value="orders" label="Copy topic" className="ml-1" />)
        const button = getByRole('button', { name: 'Copy topic' })
        const icon = button.innerHTML

        fireEvent.click(button)

        expect(writeText).toHaveBeenCalledWith('orders')
        await waitFor(() => expect(button.innerHTML).not.toBe(icon))
    })
})

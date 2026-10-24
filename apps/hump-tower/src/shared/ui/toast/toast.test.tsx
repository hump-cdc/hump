import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, render, waitFor } from '@testing-library/react'

import { Toaster, toast } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(async () => {
    act(() => {
        toast.dismiss()
    })
    cleanup()
    // Sonner removes dismissed toasts on a timer; let that work finish while the DOM still exists.
    await act(() => new Promise((resolve) => setTimeout(resolve, 300)))
})

describe('Toaster', () => {
    test('shows a toast with the kit styling and merges caller class names', async () => {
        const view = render(<Toaster style={{ zIndex: 5 }} toastOptions={{ classNames: { title: 'custom-title' } }} />)

        act(() => {
            toast.success('Saved', { description: 'All good' })
        })

        await waitFor(() => expect(view.getByText('Saved')).toBeDefined())
        expect(view.getByText('Saved').className).toContain('custom-title')
        expect(view.getByText('All good').className).toContain('text-ink-subtle!')
        const list = document.body.querySelector<HTMLElement>('[data-sonner-toaster]')
        expect(list?.dataset.yPosition).toBe('bottom')
        expect(list?.style.getPropertyValue('--normal-bg')).toBe('var(--surface-1)')
    })
})

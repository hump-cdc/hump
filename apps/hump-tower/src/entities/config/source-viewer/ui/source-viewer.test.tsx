import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, fireEvent, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { configResponse } from '@/shared/testing/fixtures.fake'
import { renderWithProviders } from '@/shared/testing/render.fake'

import { SourceViewer } from '../index'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('SourceViewer', () => {
    test('shows the config path and copies the raw configuration', async () => {
        const copied: string[] = []
        Object.defineProperty(navigator, 'clipboard', {
            configurable: true,
            value: { writeText: async (text: string) => void copied.push(text) },
        })
        fakeApi.serve({ 'GET /api/config': () => json(configResponse()) })

        const { findByText, getByLabelText, container } = renderWithProviders(<SourceViewer />)

        expect(await findByText('/etc/hump/hump.yaml')).toBeDefined()
        fireEvent.click(getByLabelText('Copy configuration'))

        await waitFor(() => expect(container.querySelector('.text-success')).not.toBeNull())
        expect(copied).toEqual(['database:\n  host: localhost\n'])
    })

    test('renders an empty source without a path when the response has neither', async () => {
        fakeApi.serve({
            'GET /api/config': () => json(configResponse({ configPath: undefined, rawConfig: undefined })),
        })

        const { getByText, queryByText } = renderWithProviders(<SourceViewer />)

        await waitFor(() => expect(fakeApi.calls()).toHaveLength(1))
        expect(getByText('read-only')).toBeDefined()
        expect(queryByText('/etc/hump/hump.yaml')).toBeNull()
    })
})

import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { configResponse } from '@/shared/testing/fixtures.fake'
import { renderWithRouter } from '@/shared/testing/render.fake'

import { Sinks } from '../index'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('Sinks', () => {
    test('lists the configured sinks and links to the operations', async () => {
        fakeApi.serve({ 'GET /api/config': () => json(configResponse()) })

        const { findByText, container } = await renderWithRouter(<Sinks />)

        expect(await findByText('2 configured')).toBeDefined()
        expect(container.querySelectorAll('.border-b')).toHaveLength(2)
        expect(container.querySelector('a[href="/operations"]')?.textContent).toContain('Open operations')
    })

    test('says so when no sink is configured', async () => {
        fakeApi.serve({ 'GET /api/config': () => json(configResponse({ sinks: null })) })

        const { findByText } = await renderWithRouter(<Sinks />)

        expect(await findByText('No sinks configured')).toBeDefined()
    })
})

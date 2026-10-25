import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { configResponse } from '@/shared/testing/fixtures.fake'
import { renderWithRouter } from '@/shared/testing/render.fake'

import { Tables } from '../index'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('Tables', () => {
    test('lists the configured tables with their sink counts and links to the flows', async () => {
        fakeApi.serve({ 'GET /api/config': () => json(configResponse()) })

        const { findByText, getByText, container } = await renderWithRouter(<Tables />)

        expect(await findByText('public.users')).toBeDefined()
        expect(getByText('1 sinks')).toBeDefined()
        expect(getByText('0 sinks')).toBeDefined()
        expect(container.querySelector('a[href="/"]')?.textContent).toContain('Open flows')
    })

    test('says so when no table is configured', async () => {
        fakeApi.serve({ 'GET /api/config': () => json(configResponse({ tables: null })) })

        const { findByText } = await renderWithRouter(<Tables />)

        expect(await findByText('No tables configured')).toBeDefined()
    })
})

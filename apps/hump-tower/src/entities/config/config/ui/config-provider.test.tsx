import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { configResponse } from '@/shared/testing/fixtures.fake'
import { renderWithProviders } from '@/shared/testing/render.fake'

import { ConfigProvider, useConfig } from '../index'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

/** Prints the loaded config path, to show the children see the resolved query. */
const ConfigPath = () => <span>{useConfig().data?.configPath}</span>

describe('ConfigProvider', () => {
    test('shows a spinner while the configuration loads, then the children', async () => {
        fakeApi.serve({ 'GET /api/config': () => json(configResponse()) })

        const { container, findByText } = renderWithProviders(
            <ConfigProvider>
                <ConfigPath />
            </ConfigProvider>,
        )

        expect(container.querySelector('.animate-spin')).not.toBeNull()
        expect(await findByText('/etc/hump/hump.yaml')).toBeDefined()
    })

    test('shows the error terminal with the status and title of a failed load', async () => {
        fakeApi.serve({
            'GET /api/config': () => json({ status: 503, title: 'Service Unavailable', detail: 'db is down' }, 503),
        })

        const { findByText, getByText } = renderWithProviders(<ConfigProvider>never</ConfigProvider>)

        expect(await findByText('Error 503: service unavailable.')).toBeDefined()
        expect(getByText('db is down')).toBeDefined()
    })

    test('falls back to a generic status, title and detail', async () => {
        fakeApi.serve({ 'GET /api/config': () => json({}, 500) })

        const { findByText, getByText } = renderWithProviders(<ConfigProvider>never</ConfigProvider>)

        expect(await findByText('Error 500: something went wrong.')).toBeDefined()
        expect(getByText('Failed to load the application configuration.')).toBeDefined()
    })

    test('requests the configuration once for every consumer', async () => {
        fakeApi.serve({ 'GET /api/config': () => json(configResponse()) })

        const { findAllByText } = renderWithProviders(
            <>
                <ConfigPath />
                <ConfigPath />
            </>,
        )

        await waitFor(async () => expect(await findAllByText('/etc/hump/hump.yaml')).toHaveLength(2))
        expect(fakeApi.calls().map(({ route }) => route)).toEqual(['GET /api/config'])
    })
})

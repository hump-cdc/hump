import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { configResponse, metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderWithProviders } from '@/shared/testing/render.fake'

import type { SelectedItem } from '../model/types'
import { InspectorDetails } from './inspector-details'

const SINK = { type: 'kafka', host: 'kafka:9092' }

const EDGE: SelectedItem = {
    kind: 'edge',
    id: 'public.users->kafka',
    sourceLabel: 'Users table',
    targetLabel: 'kafka',
    table: 'public.users',
    target: 'users',
    ...SINK,
}

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) {
        GlobalRegistrator.register({
            url: 'http://localhost/',
            settings: { device: { prefersReducedMotion: 'reduce' } },
        })
    }
    registerDom()
})
afterAll(unregisterDom)
afterEach(cleanup)

/** Serves the metrics topology and the configuration the detail views read. */
const serveBackend = (): void => {
    fakeApi.serve({
        'POST /api/metrics': () => json(metricsResponse()),
        'GET /api/config': () => json(configResponse()),
    })
}

/** Resolves once the metrics request was answered. */
const waitForMetrics = () =>
    waitFor(() => expect(fakeApi.calls().some(({ route }) => route === 'POST /api/metrics')).toBe(true))

describe('InspectorDetails', () => {
    test('shows a table with its operations and their p95', async () => {
        serveBackend()

        const view = renderWithProviders(
            <InspectorDetails selected={{ kind: 'table', id: 'public.users', name: 'public.users' }} />,
        )

        expect(view.getByText('public.users')).toBeDefined()
        for (const title of ['Total events', 'Insert', 'Update', 'Delete', 'Truncate']) {
            expect(view.getByText(title)).toBeDefined()
        }
        await waitFor(() => expect(view.container.querySelector('.font-mono')).not.toBeNull())
    })

    test('shows a sink with its type and host', async () => {
        serveBackend()

        const view = renderWithProviders(
            <InspectorDetails selected={{ kind: 'sink', id: 'kafka', name: 'Kafka', ...SINK }} />,
        )

        expect(view.getByText('Kafka')).toBeDefined()
        expect(view.getByText('kafka · kafka:9092')).toBeDefined()
        expect(view.getByText('Latency (p95)')).toBeDefined()
        await waitForMetrics()
    })

    test('shows a target with its owning sink', async () => {
        serveBackend()

        const view = renderWithProviders(
            <InspectorDetails
                selected={{ kind: 'target', id: 'kafka/users', name: 'users', sinkLabel: 'Kafka', ...SINK }}
            />,
        )

        expect(view.getByText('Kafka')).toBeDefined()
        expect(view.getByText('users')).toBeDefined()
        await waitForMetrics()
    })

    test('shows an edge with the operations its sink receives', async () => {
        serveBackend()

        const view = renderWithProviders(<InspectorDetails selected={EDGE} />)

        expect(view.getByText('Users table')).toBeDefined()
        expect(view.getByText('users')).toBeDefined()
        expect(view.getByText('kafka · kafka:9092')).toBeDefined()
        expect(await view.findByText('Operations')).toBeDefined()
        expect(view.getByText('insert')).toBeDefined()
        expect(view.getByText('update')).toBeDefined()
    })

    test('names the sink of a summary edge and omits the operations it does not filter', async () => {
        fakeApi.serve({ 'GET /api/config': () => json(configResponse()) })

        const view = renderWithProviders(
            <InspectorDetails selected={{ ...EDGE, target: undefined, table: 'public.orders' }} />,
        )

        expect(view.getByText('kafka')).toBeDefined()
        await waitFor(() => expect(fakeApi.calls().some(({ route }) => route === 'GET /api/config')).toBe(true))
        expect(view.queryByText('Operations')).toBeNull()
    })
})

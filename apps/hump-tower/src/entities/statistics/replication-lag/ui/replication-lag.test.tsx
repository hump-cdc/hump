import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderHookWithProviders, renderWithProviders } from '@/shared/testing/render.fake'

import { ReplicationLag } from '../index'
import { useReplicationLag } from '../model/use-replication-lag'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useReplicationLag', () => {
    test('reports the lag of the newest bucket', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { result } = renderHookWithProviders(() => useReplicationLag())
        expect(result.current.value).toBeUndefined()
        await waitFor(() => expect(result.current.values).toHaveLength(20))
        expect(result.current.value).toBe(1024)
        expect(result.current.values.at(-2)?.value).toBe(512)
    })
})

describe('ReplicationLag', () => {
    test('renders the Replication lag card', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { findByText } = renderWithProviders(<ReplicationLag />)

        expect(await findByText('Replication lag')).toBeDefined()
        await waitFor(() => expect(fakeApi.calls().map(({ route }) => route)).toEqual(['POST /api/metrics']))
    })
})

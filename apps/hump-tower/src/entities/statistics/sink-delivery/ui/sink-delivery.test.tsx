import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderHookWithProviders, renderWithProviders } from '@/shared/testing/render.fake'

import { SinkDelivery } from '../index'
import { useSinkDelivery } from '../model/use-sink-delivery'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useSinkDelivery', () => {
    test('charts one series per sink, cycling the colours', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { result } = renderHookWithProviders(() => useSinkDelivery())
        expect(result.current.series).toEqual([])
        await waitFor(() => expect(result.current.values).toHaveLength(20))
        expect(result.current.series.map(({ variant }) => variant)).toEqual([
            'primary',
            'success',
            'amber',
            'danger',
            'primary',
        ])
        expect(result.current.values.at(-2)).toMatchObject({ kafka: 2, http: 0 })
        expect(result.current.values.at(-1)).toMatchObject({ kafka: 1, http: 3 })
    })
})

describe('SinkDelivery', () => {
    test('renders the Sink delivery card', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { findByText } = renderWithProviders(<SinkDelivery />)

        expect(await findByText('Sink delivery')).toBeDefined()
        await waitFor(() => expect(fakeApi.calls().map(({ route }) => route)).toEqual(['POST /api/metrics']))
    })
})

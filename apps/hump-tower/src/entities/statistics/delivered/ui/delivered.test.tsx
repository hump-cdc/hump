import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { cleanup, waitFor } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { metricsResponse } from '@/shared/testing/fixtures.fake'
import { renderHookWithProviders, renderWithProviders } from '@/shared/testing/render.fake'

import { Delivered } from '../index'
import { useDelivered } from '../model/use-delivered'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useDelivered', () => {
    test('sums the delivered counts over the charted window', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { result } = renderHookWithProviders(() => useDelivered())
        expect(result.current.value).toBeUndefined()
        await waitFor(() => expect(result.current.values).toHaveLength(20))
        expect(result.current.value).toBe(7)
        expect(result.current.values.at(-1)?.value).toBe(4)
    })
})

describe('Delivered', () => {
    test('renders the Delivered card', async () => {
        fakeApi.serve({ 'POST /api/metrics': () => json(metricsResponse()) })

        const { findByText } = renderWithProviders(<Delivered />)

        expect(await findByText('Delivered')).toBeDefined()
        await waitFor(() => expect(fakeApi.calls().map(({ route }) => route)).toEqual(['POST /api/metrics']))
    })
})

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, test } from 'bun:test'

import { cleanup } from '@testing-library/react'

import { fakeApi, json } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { operationItem, operationsResponse } from '@/shared/testing/fixtures.fake'
import { fakeLiveStream } from '@/shared/testing/live.fake'
import { renderWithProviders } from '@/shared/testing/render.fake'

import type { SelectedItem } from '../model/types'
import { RecentOperations } from './recent-operations'

const TABLE: SelectedItem = { kind: 'table', id: 'public.users', name: 'public.users' }

beforeAll(registerDom)
afterAll(unregisterDom)
beforeEach(() => fakeLiveStream.reset())
afterEach(cleanup)

describe('RecentOperations', () => {
    test('lists the deliveries with their status', async () => {
        fakeApi.serve({
            'POST /api/operations': () =>
                json(
                    operationsResponse([operationItem(1), operationItem(2, { status: 'failed', operation: 'delete' })]),
                ),
        })

        const view = renderWithProviders(<RecentOperations selected={TABLE} />)

        expect(await view.findByText('INSERT')).toBeDefined()
        expect(view.getByText('DELETE')).toBeDefined()
        expect(view.container.querySelector('.bg-success')).not.toBeNull()
        expect(view.container.querySelector('.bg-danger')).not.toBeNull()
        expect(view.queryByText('No operations yet')).toBeNull()
    })

    test('says so once the selection has no deliveries', async () => {
        fakeApi.serve({ 'POST /api/operations': () => json(operationsResponse([])) })

        const view = renderWithProviders(<RecentOperations selected={TABLE} />)

        expect(view.queryByText('No operations yet')).toBeNull()
        expect(await view.findByText('No operations yet')).toBeDefined()
    })
})

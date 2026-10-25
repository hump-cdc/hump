import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, type Mock, mock, spyOn, test } from 'bun:test'

import { act, cleanup, fireEvent, waitFor } from '@testing-library/react'

import type { SnapshotDoneEventModel, SnapshotProcessingEventModel } from '@/shared/api'
import { createSseChannel, empty, fakeApi, json, type SseChannel } from '@/shared/testing/api.fake'
import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { flush, type ProvidersRender, renderWithRouter } from '@/shared/testing/render.fake'
import { toast } from '@/shared/ui/toast'

import { Snapshot } from '../index'
// Type-only module: loaded for its side effects so the coverage report lists it.
import '../model/types'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))

/** A run halfway through: one table reading, one finished without rows, one waiting. */
const processing: SnapshotProcessingEventModel = {
    state: 'processing',
    startedAt: '2026-01-01T12:00:00Z',
    tables: [
        {
            schema: 'public',
            name: 'users',
            status: 'processing',
            where: 'id > 1',
            completedRows: 50,
            estimatedRows: 100,
            sinks: [{ type: 'kafka', host: 'kafka:9092', target: 'users', count: 50 }],
        },
        { schema: 'public', name: 'orders', status: 'done', completedRows: 0, estimatedRows: 0, sinks: null },
        { schema: 'public', name: 'events', status: 'wait', completedRows: 0, estimatedRows: 10, sinks: [] },
    ],
}

/** A finished run where one table failed. */
const doneWithFailure: SnapshotDoneEventModel = {
    state: 'done',
    startedAt: '2026-01-01T12:00:00Z',
    finishedAt: '2026-01-01T12:01:30Z',
    tables: [
        {
            schema: 'public',
            name: 'users',
            status: 'success',
            error: '',
            completedRows: 100,
            sinks: [
                { type: 'kafka', host: 'kafka:9092', target: 'users', count: 100 },
                { type: 'http', host: 'api.example.com', target: '/hook', count: 60 },
            ],
        },
        { schema: 'public', name: 'orders', status: 'error', error: 'relation missing', completedRows: 3, sinks: null },
    ],
}

/** A finished run of one table, delivered to sinks that took no rows. */
const doneClean: SnapshotDoneEventModel = {
    state: 'done',
    startedAt: '2026-01-01T12:00:00Z',
    finishedAt: '2026-01-01T12:00:10Z',
    tables: [
        {
            schema: 'public',
            name: 'users',
            status: 'success',
            completedRows: 0,
            sinks: [
                { type: 'kafka', host: 'kafka:9092', target: 'users', count: 0 },
                { type: 'http', host: 'api.example.com', target: '/hook', count: 0 },
            ],
        },
    ],
}

let stream: SseChannel
let view: ProvidersRender | undefined
// Stream errors are logged; the log is muted and checked.
let warn: Mock<typeof console.warn>

beforeAll(registerDom)
afterAll(unregisterDom)

beforeEach(() => {
    warn = spyOn(console, 'warn').mockImplementation(() => {})
    stream = createSseChannel()
    fakeApi.serve({
        'GET /api/snapshot': (request) => stream.respond(request, undefined),
        'POST /api/snapshot/tables': () => json({ tables: [] }),
        'POST /api/snapshot/stop': () => empty(),
    })
})

afterEach(async () => {
    await view?.client.cancelQueries()
    view?.client.clear()
    view = undefined
    cleanup()
    stream.close()
    warn.mockRestore()
})

/** Renders the snapshot page body and waits for it to open the state stream. */
const mount = async (): Promise<ProvidersRender> => {
    view = await renderWithRouter(<Snapshot />)
    await waitFor(() => expect(stream.open()).toBe(1))
    return view
}

/** Sends `event` over the state stream and lets the page render it. */
const push = async (event: unknown): Promise<void> => {
    stream.push(event)
    await flush(20)
}

/** Requests the page sent to `route`. */
const callsTo = (route: string): number => fakeApi.calls().filter((call) => call.route === route).length

describe('Snapshot', () => {
    test('renders nothing until the stream reports a state', async () => {
        const { container } = await mount()

        expect(container.textContent).toBe('')
    })

    test('cross-fades from one state view to the next', async () => {
        const { container, findByText, queryByText } = await mount()

        await push({ state: 'idle' })
        expect(await findByText('When to use a snapshot')).toBeDefined()

        await push(processing)
        expect(await findByText('Snapshot in progress')).toBeDefined()
        const leaving = container.querySelector<HTMLElement>('[inert]')
        if (!leaving) throw new Error('The idle view did not stay mounted while fading out')

        // An animation bubbling up from inside the view does not end the fade.
        const child = leaving.querySelector('div')
        if (child) fireEvent.animationEnd(child)
        expect(queryByText('When to use a snapshot')).not.toBeNull()

        fireEvent.animationEnd(leaving)
        expect(queryByText('When to use a snapshot')).toBeNull()
    })

    test('shows the progress of a running snapshot and stops it', async () => {
        const { findByText, getByText, getAllByText } = await mount()

        await push(processing)
        expect(await findByText('Snapshot in progress')).toBeDefined()
        expect(getAllByText('public.users').length).toBeGreaterThan(0)
        expect(getAllByText('wait').length).toBeGreaterThan(0)

        await act(async () => fireEvent.click(getByText('Stop snapshot')))
        await waitFor(() => expect(callsTo('POST /api/snapshot/stop')).toBe(1))
    })

    test('reports a failed stop in a toast', async () => {
        const error = spyOn(toast, 'error').mockImplementation(() => 0)
        fakeApi.serve({
            'GET /api/snapshot': (request) => stream.respond(request, undefined),
            'POST /api/snapshot/stop': () => json({ status: 409, title: 'Conflict', detail: 'not running' }, 409),
        })
        const { findByText, getByText } = await mount()

        await push(processing)
        expect(await findByText('Snapshot in progress')).toBeDefined()
        await act(async () => fireEvent.click(getByText('Stop snapshot')))

        await waitFor(() =>
            expect(error).toHaveBeenCalledWith('Failed to stop the snapshot', { description: 'not running' }),
        )
        error.mockRestore()
    })

    test('lists the tables and sinks of a run with failures and clears it', async () => {
        const { findByText, getByText, getAllByText } = await mount()

        await push(doneWithFailure)
        expect(await findByText('Some tables were not read completely')).toBeDefined()
        expect(getByText('Snapshot is incomplete')).toBeDefined()
        expect(getByText(/2 tables/)).toBeDefined()
        expect(getAllByText('relation missing').length).toBeGreaterThan(0)
        expect(getByText('Items per sink')).toBeDefined()

        await act(async () => fireEvent.click(getByText('Run another snapshot')))
        await waitFor(() => expect(callsTo('POST /api/snapshot/stop')).toBe(1))
    })

    test('reports a clean run as complete', async () => {
        const { findByText, getByText } = await mount()

        await push(doneClean)
        expect(await findByText('Completed successfully')).toBeDefined()
        expect(getByText('Snapshot is done')).toBeDefined()
        expect(getByText(/1 table/)).toBeDefined()
    })

    test('reopens the stream after the server closes it', async () => {
        await mount()
        await push({ state: 'idle' })

        await act(async () => stream.close())
        await flush(1_100)

        expect(callsTo('GET /api/snapshot')).toBe(2)
    })

    test('logs a payload it cannot read', async () => {
        await mount()

        await push('{"state":"unknown"}')

        expect(warn).toHaveBeenCalledWith('[snapshot] stream error', expect.anything())
    })
})

import { describe, expect, mock, test } from 'bun:test'

import { Config, Snapshot } from './index'

describe('API client', () => {
    test('sends a bodyless JSON stop through the configured fetch with a base URL prefix', async () => {
        const fetch = mock(async (_input: RequestInfo | URL): Promise<Response> => new Response(null, { status: 204 }))

        const { request } = await Snapshot.stop({
            baseUrl: 'https://example.com/hump',
            fetch: Object.assign(fetch, { preconnect: (): void => {} }),
            throwOnError: true,
        })

        expect(fetch).toHaveBeenCalledTimes(1)
        expect(fetch).toHaveBeenCalledWith(request)
        expect(request.url).toBe('https://example.com/hump/api/snapshot/stop')
        expect(request.method).toBe('POST')
        expect(request.headers.get('Content-Type')).toBe('application/json')
        expect(request.body).toBeNull()
        expect(await request.text()).toBe('')
    })

    test('leaves unrelated bodyless POST and GET requests without a content type', async () => {
        const options = {
            baseUrl: 'https://example.com/hump',
            fetch: Object.assign(async (): Promise<Response> => new Response(null, { status: 204 }), {
                preconnect: (): void => {},
            }),
            throwOnError: true,
        } as const

        const tables = await Snapshot.tables(options)
        const config = await Config.get(options)

        expect(tables.request.headers.has('Content-Type')).toBe(false)
        expect(config.request.headers.has('Content-Type')).toBe(false)
    })

    test('preserves SDK problem errors from a failed stop', async () => {
        const problem = { status: 409, title: 'Conflict', detail: 'not running' }

        await expect(
            Snapshot.stop({
                baseUrl: 'https://example.com/hump',
                fetch: Object.assign(
                    async (): Promise<Response> =>
                        new Response(JSON.stringify(problem), {
                            status: 409,
                            headers: { 'Content-Type': 'application/problem+json' },
                        }),
                    { preconnect: (): void => {} },
                ),
                throwOnError: true,
            }),
        ).rejects.toEqual(problem)
    })
})

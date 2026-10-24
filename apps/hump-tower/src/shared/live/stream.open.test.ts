import { afterEach, describe, expect, spyOn, test } from 'bun:test'

import { createLiveStream } from './stream'
import type { LiveStatus } from './types'
// Type-only module: imported for its side effects alone so the coverage report lists it.
import './types'

// The default `open` (the generated `Live.stream`) and the end of a run that `stop` did not cause.
// `stream.test.ts` injects `open` everywhere else.

const originalAbortController = globalThis.AbortController

afterEach(() => {
    globalThis.AbortController = originalAbortController
})

describe('createLiveStream with the default transport', () => {
    test('logs a failed attempt through onSseError and goes on reconnecting', async () => {
        const warn = spyOn(console, 'warn').mockImplementation(() => {})
        const sleeps: number[] = []
        const stream = createLiveStream({
            sleep: async (ms) => {
                sleeps.push(ms)
                stream.stop()
            },
        })

        try {
            stream.retain()
            await until(() => sleeps.length > 0)

            expect(warn.mock.calls.some(([message]) => message === '[live] stream error')).toBe(true)
            expect(stream.getStatus()).toBe('idle')
        } finally {
            stream.stop()
            warn.mockRestore()
        }
    })
})

describe('createLiveStream run lifecycle', () => {
    test('drops to idle when its signal aborts without a stop', async () => {
        const controllers: AbortController[] = []
        globalThis.AbortController = class extends originalAbortController {
            constructor() {
                super()
                controllers.push(this)
            }
        }
        const statuses: LiveStatus[] = []
        const stream = createLiveStream({
            open: (signal) =>
                Promise.resolve({
                    async *[Symbol.asyncIterator]() {
                        await new Promise((resolve) => signal.addEventListener('abort', resolve, { once: true }))
                    },
                }),
            lingerMs: 0,
        })
        stream.subscribeStatus(() => statuses.push(stream.getStatus()))

        stream.retain()
        await until(() => stream.getStatus() === 'connecting')
        controllers[0].abort()
        await until(() => stream.getStatus() === 'idle')

        expect(statuses).toEqual(['connecting', 'idle'])
    })
})

/** Polls `condition` on zero-delay timers until it holds, failing after a second. */
const until = async (condition: () => boolean): Promise<void> => {
    const deadline = Date.now() + 1_000
    while (!condition()) {
        if (Date.now() > deadline) throw new Error('condition not met in time')
        await new Promise((resolve) => setTimeout(resolve, 0))
    }
}

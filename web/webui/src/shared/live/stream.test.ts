import { describe, expect, test } from 'bun:test'

import type { GetApiLiveResponse } from '@/shared/api'

import { STABLE_ATTEMPT_MS } from './backoff'
import { createLiveStream, type LiveStream } from './stream'
import type { LiveOperationsBatch, LiveStatus } from './types'

const OPERATION = { schema: 'public', table: 'users', sink: 'kafka', host: 'localhost', target: 'users' }

/** One attempt's event source, driven by the test. */
type Channel = {
    iterable: AsyncIterable<GetApiLiveResponse>
    push: (event: GetApiLiveResponse) => void
    end: () => void
    fail: (error: unknown) => void
}

const createChannel = (): Channel => {
    const queue: GetApiLiveResponse[] = []
    let wake: (() => void) | undefined
    let done = false
    let failure: unknown

    const notify = (): void => {
        wake?.()
        wake = undefined
    }

    return {
        iterable: {
            async *[Symbol.asyncIterator]() {
                while (true) {
                    while (queue.length > 0) {
                        yield queue.shift() as GetApiLiveResponse
                    }
                    if (failure !== undefined) {
                        throw failure
                    }
                    if (done) {
                        return
                    }
                    await new Promise<void>((resolve) => {
                        wake = resolve
                    })
                }
            },
        },
        push: (event) => {
            queue.push(event)
            notify()
        },
        end: () => {
            done = true
            notify()
        },
        fail: (error) => {
            failure = error
            notify()
        },
    }
}

type Harness = {
    stream: LiveStream
    attempts: Channel[]
    sleeps: number[]
    advance: (ms: number) => void
}

const setup = (lingerMs = 0): Harness => {
    const attempts: Channel[] = []
    const sleeps: number[] = []
    let clock = 0

    const stream = createLiveStream({
        open: async (signal) => {
            const channel = createChannel()
            signal.addEventListener('abort', () => channel.end(), { once: true })
            attempts.push(channel)
            return channel.iterable
        },
        sleep: async (ms) => {
            sleeps.push(ms)
            clock += ms
        },
        now: () => clock,
        lingerMs,
    })

    return {
        stream,
        attempts,
        sleeps,
        advance: (ms) => {
            clock += ms
        },
    }
}

/** Lets pending microtasks and zero-delay timers run. */
const tick = (times = 4): Promise<void> =>
    Array.from({ length: times }).reduce<Promise<void>>(
        (chain) => chain.then(() => new Promise<void>((resolve) => setTimeout(resolve, 0))),
        Promise.resolve(),
    )

const delay = (ms: number): Promise<void> => new Promise((resolve) => setTimeout(resolve, ms))

const event = (count: number): GetApiLiveResponse => ({
    operations: Array.from({ length: count }, () => ({ ...OPERATION })),
})

describe('createLiveStream fan-out', () => {
    test('delivers a batch to every listener exactly once', async () => {
        const { stream, attempts } = setup()
        const seen: LiveOperationsBatch[] = []
        const other: LiveOperationsBatch[] = []

        stream.retain()
        stream.subscribe((batch) => seen.push(batch))
        stream.subscribe((batch) => other.push(batch))
        await tick()

        attempts[0].push(event(2))
        await tick()

        expect(seen).toHaveLength(1)
        expect(seen[0]).toHaveLength(2)
        expect(other).toHaveLength(1)

        stream.stop()
    })

    test('a throwing listener neither stops the stream nor starves the others', async () => {
        const { stream, attempts } = setup()
        const seen: LiveOperationsBatch[] = []

        stream.retain()
        stream.subscribe(() => {
            throw new Error('boom')
        })
        stream.subscribe((batch) => seen.push(batch))
        await tick()

        attempts[0].push(event(1))
        attempts[0].push(event(1))
        await tick()

        expect(seen).toHaveLength(2)

        stream.stop()
    })

    test('unsubscribing from inside a listener does not break the iteration', async () => {
        const { stream, attempts } = setup()
        const seen: string[] = []

        stream.retain()
        const off = stream.subscribe(() => {
            seen.push('first')
            off()
        })
        stream.subscribe(() => seen.push('second'))
        await tick()

        attempts[0].push(event(1))
        await tick()

        expect(seen).toEqual(['first', 'second'])

        stream.stop()
    })

    test('an empty or null batch never reaches listeners', async () => {
        const { stream, attempts } = setup()
        const seen: LiveOperationsBatch[] = []

        stream.retain()
        stream.subscribe((batch) => seen.push(batch))
        await tick()

        attempts[0].push({ operations: [] })
        attempts[0].push({ operations: null } as unknown as GetApiLiveResponse)
        await tick()

        expect(seen).toHaveLength(0)

        stream.stop()
    })
})

describe('createLiveStream reconnection', () => {
    test('reopens after a clean end of the stream', async () => {
        const { stream, attempts } = setup()

        stream.retain()
        await tick()
        expect(attempts).toHaveLength(1)

        attempts[0].end()
        await tick()

        expect(attempts).toHaveLength(2)

        stream.stop()
    })

    test('reopens after a failed attempt', async () => {
        const { stream, attempts } = setup()

        stream.retain()
        await tick()

        attempts[0].fail(new Error('dropped'))
        await tick()

        expect(attempts).toHaveLength(2)

        stream.stop()
    })

    test('notifies reconnect listeners from the second attempt on', async () => {
        const { stream, attempts } = setup()
        let reconnects = 0

        stream.retain()
        stream.subscribeReconnect(() => {
            reconnects += 1
        })
        await tick()

        expect(reconnects).toBe(0)

        attempts[0].end()
        await tick()
        expect(reconnects).toBe(1)

        attempts[1].end()
        await tick()
        expect(reconnects).toBe(2)

        stream.stop()
    })

    test('a short attempt grows the backoff', async () => {
        const { stream, attempts, sleeps } = setup()

        stream.retain()
        await tick()

        attempts[0].end()
        await tick()
        attempts[1].end()
        await tick()

        expect(sleeps).toEqual([1_000, 2_000])

        stream.stop()
    })

    test('an attempt that lived long enough resets the backoff', async () => {
        const { stream, attempts, sleeps, advance } = setup()

        stream.retain()
        await tick()

        attempts[0].end()
        await tick()

        advance(STABLE_ATTEMPT_MS)
        attempts[1].end()
        await tick()

        expect(sleeps).toEqual([1_000, 1_000])

        stream.stop()
    })

    test('stop ends the loop and does not reopen', async () => {
        const { stream, attempts } = setup()

        stream.retain()
        await tick()

        stream.stop()
        await tick()

        expect(attempts).toHaveLength(1)
        expect(stream.getStatus()).toBe('idle')
    })
})

describe('createLiveStream status', () => {
    test('walks idle, connecting, live, reconnecting, connecting', async () => {
        const { stream, attempts } = setup()
        const seen: LiveStatus[] = []

        stream.subscribeStatus(() => seen.push(stream.getStatus()))
        expect(stream.getStatus()).toBe('idle')

        stream.retain()
        await tick()

        attempts[0].push(event(1))
        await tick()

        attempts[0].end()
        await tick()

        expect(seen).toEqual(['connecting', 'live', 'reconnecting', 'connecting'])

        stream.stop()
    })

    test('goes live on the welcome event, before any delivery', async () => {
        const { stream, attempts } = setup()

        stream.retain()
        await tick()
        expect(stream.getStatus()).toBe('connecting')

        attempts[0].push({})
        await tick()

        expect(stream.getStatus()).toBe('live')

        stream.stop()
    })
})

describe('createLiveStream retain', () => {
    test('reuses the connection when a release is followed by a retain inside the linger', async () => {
        const { stream, attempts } = setup(50)

        const release = stream.retain()
        await tick()
        release()
        stream.retain()

        await delay(80)

        expect(attempts).toHaveLength(1)
        expect(stream.getStatus()).not.toBe('idle')

        stream.stop()
    })

    test('stops once the last hold is released and the linger elapses', async () => {
        const { stream } = setup(20)

        const release = stream.retain()
        await tick()
        release()

        expect(stream.getStatus()).not.toBe('idle')

        await delay(60)

        expect(stream.getStatus()).toBe('idle')
    })

    test('a release called twice does not drop another hold', async () => {
        const { stream } = setup(20)

        const first = stream.retain()
        stream.retain()
        await tick()

        first()
        first()

        await delay(60)

        expect(stream.getStatus()).not.toBe('idle')

        stream.stop()
    })
})

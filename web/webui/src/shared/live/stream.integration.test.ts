import { describe, expect, test } from 'bun:test'

import { Live, type OperationsEventModel } from '@/shared/api'

const OPERATION = { schema: 'public', table: 'users', sink: 'kafka', host: 'localhost', target: 'users' }

/** Serves `frames` as SSE and closes, so a test sees both delivery and a clean end of stream. */
const serveFrames = (frames: string[]): ReturnType<typeof Bun.serve> =>
    Bun.serve({
        port: 0,
        fetch: () =>
            new Response(
                new ReadableStream({
                    start(controller) {
                        const encoder = new TextEncoder()
                        for (const frame of frames) {
                            controller.enqueue(encoder.encode(frame))
                        }
                        controller.close()
                    },
                }),
                { headers: { 'content-type': 'text/event-stream' } },
            ),
    })

const frame = (operations: object[]): string => `event: operations\ndata: ${JSON.stringify({ operations })}\n\n`

describe('Live.stream against a real server', () => {
    test('yields parsed events and ends when the server closes the stream', async () => {
        const server = serveFrames([frame([OPERATION]), frame([OPERATION, OPERATION])])

        try {
            const { stream } = await Live.stream({
                baseUrl: server.url.origin,
                sseMaxRetryAttempts: 1,
            })

            const received: OperationsEventModel[] = []
            for await (const event of stream) {
                received.push(event)
            }

            expect(received).toHaveLength(2)
            expect(received[0].operations).toEqual([OPERATION])
            expect(received[1].operations).toHaveLength(2)
        } finally {
            await server.stop(true)
        }
    })

    test('drops a payload the event schema rejects and ends the attempt', async () => {
        const server = serveFrames([`event: operations\ndata: ${JSON.stringify({ operations: [{}] })}\n\n`])
        const errors: unknown[] = []

        try {
            const { stream } = await Live.stream({
                baseUrl: server.url.origin,
                sseMaxRetryAttempts: 1,
                onSseError: (error) => errors.push(error),
            })

            const received: OperationsEventModel[] = []
            for await (const event of stream) {
                received.push(event)
            }

            expect(received).toHaveLength(0)
            expect(errors).toHaveLength(1)
        } finally {
            await server.stop(true)
        }
    })
})

/** Answers one API request; `body` is the parsed JSON request body, if any. */
export type ApiHandler = (request: Request, body: unknown) => Response | Promise<Response>

/** Handlers keyed by `<METHOD> <path>`, e.g. `POST /api/metrics`. */
export type ApiRoutes = Record<string, ApiHandler>

/** Requests the fake API received, in order, as `<METHOD> <path>` with the parsed body. */
export type ApiCall = { route: string; body: unknown }

/** A server-sent event stream the test pushes events into while the client listens. */
export type SseChannel = {
    /** The response for one request; the stream stays open until `close` or an abort. */
    respond: ApiHandler
    /** Sends `event` to every open stream. */
    push: (event: unknown) => void
    /** Ends every open stream, as a server closing the connection. */
    close: () => void
    /** Streams open right now. */
    open: () => number
}

/**
 * The fake backend the generated client talks to: `globalThis.fetch` routed by method and path.
 * An unknown route answers 404, so a forgotten handler fails the query instead of hanging.
 * `registerDom` installs it.
 */
export const fakeApi = (() => {
    let routes: ApiRoutes = {}
    const calls: ApiCall[] = []

    const handle = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
        const request = input instanceof Request ? input : new Request(input, init)
        const route = `${request.method} ${new URL(request.url).pathname}`
        const text = await request.text()
        const body: unknown = text ? JSON.parse(text) : undefined
        calls.push({ route, body })

        const handler = routes[route]
        return handler ? handler(request, body) : json({ status: 404, title: 'Not Found' }, 404)
    }

    return {
        /** Replaces the handlers and forgets the recorded calls. */
        serve: (next: ApiRoutes): void => {
            routes = next
            calls.length = 0
        },
        /** Routes of the requests received so far. */
        calls: (): ApiCall[] => [...calls],
        fetch: handle,
    }
})()

/** A JSON response with the given status. */
export const json = (body: unknown, status = 200): Response =>
    new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } })

/** An empty response, as the start and stop endpoints answer. */
export const empty = (status = 204): Response => new Response(null, { status })

/**
 * A server-sent event stream of `events`, one `data:` frame each. It closes after the last frame
 * unless `open` is set, in which case it stays open until the request is aborted.
 */
export const sse = (request: Request, events: readonly unknown[], open = false): Response =>
    new Response(
        new ReadableStream<Uint8Array>({
            start(controller) {
                const encoder = new TextEncoder()
                for (const event of events) {
                    const data = typeof event === 'string' ? event : JSON.stringify(event)
                    controller.enqueue(encoder.encode(`data: ${data}\n\n`))
                }
                if (!open) {
                    controller.close()
                    return
                }
                request.signal.addEventListener('abort', () => controller.close())
            },
        }),
        { headers: { 'content-type': 'text/event-stream' } },
    )

/** Creates an `SseChannel` with no stream open yet. */
export const createSseChannel = (): SseChannel => {
    const controllers = new Set<ReadableStreamDefaultController<Uint8Array>>()
    const encoder = new TextEncoder()

    return {
        respond: (request) =>
            new Response(
                new ReadableStream<Uint8Array>({
                    start(controller) {
                        controllers.add(controller)
                        request.signal.addEventListener('abort', () => {
                            if (controllers.delete(controller)) controller.close()
                        })
                    },
                }),
                { headers: { 'content-type': 'text/event-stream' } },
            ),
        push: (event) => {
            const data = typeof event === 'string' ? event : JSON.stringify(event)
            for (const controller of controllers) controller.enqueue(encoder.encode(`data: ${data}\n\n`))
        },
        close: () => {
            for (const controller of controllers) controller.close()
            controllers.clear()
        },
        open: () => controllers.size,
    }
}

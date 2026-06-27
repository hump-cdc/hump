/// <reference lib="webworker" />

import { type EventModel, Operations, type RetryEventModel } from '@/shared/api'
import { sleep } from '@/shared/utils'
import { runWorker, type WorkerContext } from '@/shared/workers/worker'

import { StreamBuffer } from './buffer'

const OPERATIONS_LOCK = 'hump:operations-stream'

const buffer = new StreamBuffer()

const run = async ({ signal }: WorkerContext): Promise<void> => {
    while (!signal.aborted) {
        try {
            const { stream } = await Operations.stream({
                signal,
                responseTransformer: undefined,
                onSseError: (error) => console.error('[sse:error]', error),
            })

            for await (const message of stream) {
                const event = message as unknown as EventModel | RetryEventModel
                if ('attempt' in event) {
                    buffer.addRetry(event)
                    continue
                }

                buffer.add(event)
            }
        } catch (error) {
            if (signal.aborted) {
                break
            }
            console.error('[sse:error]', error)
        }

        if (signal.aborted) break

        await sleep(500, signal)
    }
}

runWorker({
    lock: OPERATIONS_LOCK,
    label: 'operations',
    run,
    onStop: () => buffer.stop(),
})

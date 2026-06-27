/// <reference lib="webworker" />

import { sleep } from '@/shared/utils'
import { runWorker, type WorkerContext } from '@/shared/workers/worker'

import { persist } from './persist'
import { prime } from './prime'
import { scrape } from './scrape'

const CYCLE_MS = 5_000
const TARGET_MS = 9_000
const METRICS_LOCK = 'hump:metrics'

const nextDelay = (): number => {
    const delay = (TARGET_MS - (Date.now() % CYCLE_MS) + CYCLE_MS) % CYCLE_MS
    return delay === 0 ? CYCLE_MS : delay
}

const run = async ({ signal }: WorkerContext): Promise<void> => {
    // Baseline the first scrape so its lifetime counters never reach the series.
    try {
        const metrics = await scrape(signal)
        if (!metrics) return

        await prime(metrics)
    } catch (error) {
        if (!signal.aborted) console.error('[metrics:prime]', error)
    }

    // Poll on the cycle until the page stops us.
    while (!signal.aborted) {
        await sleep(nextDelay(), signal)
        if (signal.aborted) break

        try {
            const metrics = await scrape(signal)
            const now = Date.now()
            if (!metrics) continue

            await persist(metrics, now)
        } catch (error) {
            if (signal.aborted) break

            console.error('[metrics:error]', error)
        }
    }
}

runWorker({ lock: METRICS_LOCK, label: 'metrics', run })

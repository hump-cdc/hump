import { useCallback, useSyncExternalStore } from 'react'

import {
    applyGlobal,
    applyLocal,
    type LiveStreamMessage,
    openLiveStreamChannel,
    rememberedRunning,
} from './broadcast-channel'
import { WORKERS } from './workers'

/**
 * The dashboard's live feed as one switch over both workers and every open tab.
 * `running` and `watch` are shaped for `useSyncExternalStore`.
 */
export const liveStream = {
    /** Start the feed in every open tab. */
    start: (): void => applyGlobal(true),

    /** Stop the feed in every open tab, leaving the threads up for a restart. */
    stop: (): void => applyGlobal(false),

    /** Live only while every worker is running. */
    running: (): boolean => WORKERS.every((worker) => worker.running),

    /** Subscribe to any worker's start/stop. Returns an unsubscribe function. */
    watch: (listener: () => void): (() => void) => {
        const unwatch = WORKERS.map((worker) => worker.watch(listener))
        return () => {
            for (const off of unwatch) off()
        }
    },

    /**
     * Adopt the remembered choice and follow the other tabs while mounted. Teardown stops
     * this tab's workers only — unmounting is not the user pressing Stop.
     */
    attach: (): (() => void) => {
        const listener = (event: MessageEvent<LiveStreamMessage>) => {
            if (event.data.type === 'live-stream') applyLocal(event.data.running)
        }
        openLiveStreamChannel()?.addEventListener('message', listener)

        applyLocal(rememberedRunning())

        return () => {
            openLiveStreamChannel()?.removeEventListener('message', listener)
            applyLocal(false)
        }
    },
}

/** Read and drive the live feed. `running` comes from the worker managers themselves, so a control cannot disagree with them. */
export const useLiveStream = (): {
    running: boolean
    start: () => void
    stop: () => void
    toggle: () => void
} => {
    const running = useSyncExternalStore(liveStream.watch, liveStream.running)

    const toggle = useCallback(() => {
        if (running) liveStream.stop()
        else liveStream.start()
    }, [running])

    return { running, start: liveStream.start, stop: liveStream.stop, toggle }
}

import { WORKERS } from './workers'

/** Both the cross-tab channel name and the key remembering the last choice. */
const LIVE_STREAM_KEY = 'hump.live-stream'

/** What a tab tells the others when the user starts or stops the feed. */
export type LiveStreamMessage = { type: 'live-stream'; running: boolean }

/** Where the workers read from: the configured API, or whatever origin served the page. */
const baseUrl = (): string => API_URL || globalThis.location.origin

// One lazily opened channel per name, kept until the tab goes away.
const channels = new Map<string, BroadcastChannel>()

/** The channel for `name`, or null where the browser has no `BroadcastChannel`. */
const openChannel = (name: string): BroadcastChannel | null => {
    const open = channels.get(name)
    if (open) return open

    if (typeof BroadcastChannel === 'undefined') return null

    const channel = new BroadcastChannel(name)
    channels.set(name, channel)
    return channel
}

/** The channel the tabs trade start/stop over. */
export const openLiveStreamChannel = (): BroadcastChannel | null => openChannel(LIVE_STREAM_KEY)

/** The last start/stop chosen in any tab — what a tab opened later inherits. */
export const rememberedRunning = (): boolean => {
    try {
        return localStorage.getItem(LIVE_STREAM_KEY) !== 'false'
    } catch {
        // localStorage may be blocked (private mode) — default to live.
        return true
    }
}

const rememberRunning = (running: boolean): void => {
    try {
        localStorage.setItem(LIVE_STREAM_KEY, String(running))
    } catch {
        // Ignore: the choice still reaches the tabs already open.
    }
}

/** Bring this tab's workers in line, without telling any other tab. */
export const applyLocal = (running: boolean): void => {
    for (const worker of WORKERS) {
        if (running) worker.start(baseUrl())
        else worker.stop()
    }
}

/** Apply the user's choice here, remember it, and hand it to every other tab. */
export const applyGlobal = (running: boolean): void => {
    rememberRunning(running)
    applyLocal(running)
    openLiveStreamChannel()?.postMessage({
        type: 'live-stream',
        running,
    } satisfies LiveStreamMessage)
}

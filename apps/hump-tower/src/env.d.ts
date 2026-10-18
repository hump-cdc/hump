declare const VERSION: string
declare const API_URL: string

interface ImportMeta {
    /** Only present under the dev server; used to close long-lived connections on module replacement. */
    readonly webpackHot?: { dispose: (callback: () => void) => void }
}

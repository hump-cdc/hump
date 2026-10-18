/** Prefix the sinks put headers under in an operation's metadata. */
const HEADER_PREFIX = 'header.'

/** One metadata entry as the details list shows it. */
export type Header = {
    key: string
    value: string
}

/** Lists every metadata entry in its original order, stripping the `header.` prefix from headers. */
export const toHeaders = (metadata: Record<string, string>): Header[] =>
    Object.entries(metadata).map(([key, value]) => ({
        key: key.startsWith(HEADER_PREFIX) ? key.slice(HEADER_PREFIX.length) : key,
        value,
    }))

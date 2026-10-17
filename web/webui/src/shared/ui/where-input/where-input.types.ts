import type { WhereCompletion } from './where-input.completions'

/** Open completion list, anchored to the caret it was computed at. */
export type Suggestions = WhereCompletion & {
    active: number
    anchor: { getBoundingClientRect: () => DOMRect }
}

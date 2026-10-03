import type { PrismEditor } from 'prism-react-editor'
import { highlightText } from 'prism-react-editor/prism'

import { BORDER, LINE_HEIGHT, MAX_ROWS, MIN_ROWS, VERTICAL_PADDING } from './where-input.consts'
import { WHERE_LANGUAGE } from './where-input.grammar'

export const clampHeight = (contentHeight: number) =>
    Math.min(
        Math.max(contentHeight, MIN_ROWS * LINE_HEIGHT + VERTICAL_PADDING),
        MAX_ROWS * LINE_HEIGHT + VERTICAL_PADDING,
    ) + BORDER

/**
 * Natural content height of the editor. The wrapper stretches to fill the editor's grid row
 * (so the textarea covers the whole box), hence measure to the bottom of the last line instead.
 */
export const contentHeightOf = (editor: PrismEditor) => {
    const lines = editor.lines
    const last = lines?.[lines.length - 1]
    return last ? last.offsetTop + last.offsetHeight + VERTICAL_PADDING : 0
}

/** Virtual anchor at the editor's caret, read when the popup positions itself. */
export const caretAnchor = (editor: PrismEditor) => ({
    getBoundingClientRect: () => {
        const overlays = editor.lines?.[0]
        const caret = editor.extensions.cursor?.getPosition()
        if (!overlays || !caret) return new DOMRect()
        const rect = overlays.getBoundingClientRect()
        return new DOMRect(rect.left + caret.left, rect.top + caret.top, 0, caret.height)
    },
})

/**
 * Highlighted first non-blank line shown while collapsed, and whether more lines follow it.
 * `null` for a blank value, so the placeholder shows instead.
 */
export const collapsedPreview = (value: string) => {
    const lines = value.split('\n')
    const first = lines.findIndex((line) => line.trim())
    if (first < 0) return null
    return {
        html: highlightText(lines[first], WHERE_LANGUAGE),
        more: lines.slice(first + 1).some((line) => line.trim()),
    }
}

import type { FC } from 'react'

import { useCursorPosition } from 'prism-react-editor/cursor'
import { usePrismEditor } from 'prism-react-editor/extensions'

/** Editor extension tracking the caret position, used to anchor the suggestion popup. */
export const CaretTracker: FC = () => {
    const [editor] = usePrismEditor()
    useCursorPosition(editor)
    return null
}

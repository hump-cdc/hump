import { useMemo } from 'react'

import { useFlowInspectorStore } from '@/entities/flows/inspector'
import type { ViewportInsets } from '@/shared/ui/flow'
import { useElementSize, useIsMobile } from '@/shared/utils'

/** Desktop inspector width (px), matching its `w-75`. */
const INSPECTOR_WIDTH = 300

/**
 * Canvas margins covered by the floating toolbar (top) and the inspector (right on desktop,
 * bottom on mobile). Attach `toolbarRef`/`inspectorRef` to measure them; `isShrunk` tells whether
 * the desktop inspector is open and the toolbar should yield its width.
 */
export const useInsets = (): {
    toolbarRef: (element: HTMLDivElement | null) => void
    inspectorRef: (element: HTMLDivElement | null) => void
    isShrunk: boolean
    viewportInsets: ViewportInsets
} => {
    const isInspectorOpen = useFlowInspectorStore((state) => state.selected !== null)
    const isMobile = useIsMobile()

    const [toolbarRef, { height: toolbarHeight }] = useElementSize<HTMLDivElement>()
    const [inspectorRef, { height: inspectorHeight }] = useElementSize<HTMLDivElement>()

    // On desktop the floating toolbar yields the inspector's width (300px), in step with the
    // sheet's open (300ms) and close (200ms) animations. On mobile the inspector opens from the bottom.
    const isShrunk = isInspectorOpen && !isMobile

    // The desktop inspector width is known up front, so it changes in the same render as the
    // selection it is revealed for; the mobile sheet's height depends on its content and is measured.
    const insetRight = isShrunk ? INSPECTOR_WIDTH : 0
    const insetBottom = isInspectorOpen && isMobile ? inspectorHeight : 0
    const viewportInsets = useMemo(
        () => ({ top: toolbarHeight, right: insetRight, bottom: insetBottom, left: 0 }),
        [toolbarHeight, insetRight, insetBottom],
    )

    return { toolbarRef, inspectorRef, isShrunk, viewportInsets }
}

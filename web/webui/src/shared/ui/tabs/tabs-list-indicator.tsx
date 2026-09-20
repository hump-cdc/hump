import { type FC, type RefObject, useCallback, useEffect, useRef, useState } from 'react'

import { cn } from '@/shared/utils'

type TabsListIndicatorProps = {
    listRef: RefObject<HTMLDivElement | null>
    /** `pill` covers the active trigger; `underline` draws a bar along its bottom edge. */
    variant: 'pill' | 'underline'
}

/** Thickness (px) of the `underline` indicator. */
const UNDERLINE_HEIGHT = 2

export const TabsListIndicator: FC<TabsListIndicatorProps> = ({ listRef, variant }) => {
    const indicatorRef = useRef<HTMLDivElement>(null)
    const [ready, setReady] = useState(false)

    const updateIndicator = useCallback(() => {
        const list = listRef.current
        const indicator = indicatorRef.current
        if (!list || !indicator) return

        const activeTrigger = list.querySelector<HTMLElement>('[data-slot="tabs-trigger"][data-state="active"]')
        if (!activeTrigger) {
            indicator.style.opacity = '0'
            return
        }

        indicator.style.opacity = '1'
        indicator.style.left = `${activeTrigger.offsetLeft}px`
        indicator.style.width = `${activeTrigger.offsetWidth}px`
        if (variant === 'underline') {
            indicator.style.top = `${activeTrigger.offsetTop + activeTrigger.offsetHeight - UNDERLINE_HEIGHT}px`
            indicator.style.height = `${UNDERLINE_HEIGHT}px`
        } else {
            indicator.style.top = `${activeTrigger.offsetTop}px`
            indicator.style.height = `${activeTrigger.offsetHeight}px`
        }

        if (!ready) setReady(true)
    }, [listRef, ready, variant])

    useEffect(() => {
        const list = listRef.current
        if (!list) return

        updateIndicator()

        const observer = new MutationObserver(updateIndicator)
        observer.observe(list, { attributes: true, attributeFilter: ['data-state'], subtree: true, childList: true })

        // Triggers resize with the list (e.g. full-width tabs across viewport changes), so re-measure on resize too.
        const resizeObserver = new ResizeObserver(updateIndicator)
        resizeObserver.observe(list)

        return () => {
            observer.disconnect()
            resizeObserver.disconnect()
        }
    }, [listRef, updateIndicator])

    return (
        <div
            ref={indicatorRef}
            className={cn(
                'pointer-events-none absolute select-none',
                variant === 'underline'
                    ? 'bg-primary'
                    : 'rounded-md border border-hairline-strong bg-canvas shadow-sm dark:bg-surface-3',
                ready && 'transition-all duration-300 ease-out',
            )}
        />
    )
}

import { type FC, type RefObject, useCallback, useEffect, useRef, useState } from 'react'

import { cn } from '@/shared/utils'

type TabsListIndicatorProps = {
    listRef: RefObject<HTMLDivElement | null>
}

export const TabsListIndicator: FC<TabsListIndicatorProps> = ({ listRef }) => {
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
        indicator.style.top = `${activeTrigger.offsetTop}px`
        indicator.style.width = `${activeTrigger.offsetWidth}px`
        indicator.style.height = `${activeTrigger.offsetHeight}px`

        if (!ready) setReady(true)
    }, [listRef, ready])

    useEffect(() => {
        const list = listRef.current
        if (!list) return

        updateIndicator()

        const observer = new MutationObserver(updateIndicator)
        observer.observe(list, { attributes: true, attributeFilter: ['data-state'], subtree: true, childList: true })

        return () => observer.disconnect()
    }, [listRef, updateIndicator])

    return (
        <div
            ref={indicatorRef}
            className={cn(
                'pointer-events-none absolute select-none rounded-md border border-hairline-strong bg-canvas shadow-sm dark:bg-surface-3',
                ready && 'transition-all duration-300 ease-out',
            )}
        />
    )
}

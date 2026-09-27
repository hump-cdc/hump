import { useCallback, useEffect, useState } from 'react'

type ElementSize = { width: number; height: number }

const ZERO_SIZE: ElementSize = { width: 0, height: 0 }

/**
 * Track an element's border-box size: attach the returned callback ref, read the size. Reports
 * zero while nothing is attached (e.g. after the element unmounts).
 */
export const useElementSize = <T extends HTMLElement>(): [(element: T | null) => void, ElementSize] => {
    const [element, setElement] = useState<T | null>(null)
    const [size, setSize] = useState<ElementSize>(ZERO_SIZE)

    const ref = useCallback((node: T | null) => setElement(node), [])

    useEffect(() => {
        if (!element) {
            setSize(ZERO_SIZE)
            return
        }

        const update = () =>
            setSize((prev) =>
                prev.width === element.offsetWidth && prev.height === element.offsetHeight
                    ? prev
                    : { width: element.offsetWidth, height: element.offsetHeight },
            )
        update()

        const observer = new ResizeObserver(update)
        observer.observe(element)
        return () => observer.disconnect()
    }, [element])

    return [ref, size]
}

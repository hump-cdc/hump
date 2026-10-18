import { useEffect, useState } from 'react'

export const MOBILE_BREAKPOINT = 1024

const query = (breakpoint: number): string => `(max-width: ${breakpoint - 1}px)`

const matches = (breakpoint: number): boolean =>
    typeof window != 'undefined' && window.matchMedia(query(breakpoint)).matches

export const useIsMobile = (breakpoint: number = MOBILE_BREAKPOINT): boolean => {
    const [isMobile, setIsMobile] = useState<boolean>(() => matches(breakpoint))

    useEffect(() => {
        const media = window.matchMedia(query(breakpoint))
        const onChange = () => setIsMobile(media.matches)
        onChange()
        media.addEventListener('change', onChange)
        return () => media.removeEventListener('change', onChange)
    }, [breakpoint])

    return isMobile
}

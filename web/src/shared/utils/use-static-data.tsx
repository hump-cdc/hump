import type { StaticDataRouteOption } from '@tanstack/react-router'
import { useMatches } from '@tanstack/react-router'

import type { ReactNode } from 'react'

declare module '@tanstack/react-router' {
    interface StaticDataRouteOption {
        title?: string
        headerAside?: ReactNode
    }
}

export const useStaticData = (): StaticDataRouteOption => {
    const matches = useMatches()
    return [...matches].reverse().find((match) => match.staticData.title)?.staticData ?? {}
}

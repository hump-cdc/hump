import type { ComponentType, ReactNode } from 'react'
import { createContext, useContext } from 'react'

export const THEMES = { light: '', dark: '.dark' } as const

export type ChartConfig = Record<
    string,
    {
        label?: ReactNode
        icon?: ComponentType
    } & ({ color?: string; theme?: never } | { color?: never; theme: Record<keyof typeof THEMES, string> })
>

type ChartContextProps = {
    config: ChartConfig
}

export const ChartContext = createContext<ChartContextProps | null>(null)

export const useChart = () => {
    const context = useContext(ChartContext)

    if (!context) {
        throw new Error('useChart must be used within a <ChartContainer />')
    }

    return context
}

import { useContext } from 'react'

import type { ThemeContextValue } from './theme-provider'
import { ThemeContext } from './theme-provider'

export const useTheme = (): ThemeContextValue => {
    const context = useContext(ThemeContext)
    if (!context) throw new Error('useTheme must be used within a <ThemeProvider>')
    return context
}

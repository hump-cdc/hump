import type { FC, PropsWithChildren } from 'react'

import { ThemeProvider } from '@/shared/theme'
import { TooltipProvider } from '@/shared/ui/tooltip'

type ProvidersProps = PropsWithChildren

export const Providers: FC<ProvidersProps> = ({ children }) => (
    <ThemeProvider>
        <TooltipProvider>{children}</TooltipProvider>
    </ThemeProvider>
)

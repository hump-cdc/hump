import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import type { FC, PropsWithChildren } from 'react'

import { ConfigProvider } from '@/entities/config/config'
import { ThemeProvider } from '@/shared/theme'
import { TooltipProvider } from '@/shared/ui/tooltip'

const queryClient = new QueryClient()

type ProvidersProps = PropsWithChildren

export const Providers: FC<ProvidersProps> = ({ children }) => (
    <QueryClientProvider client={queryClient}>
        <ThemeProvider>
            <TooltipProvider>
                <ConfigProvider>{children}</ConfigProvider>
            </TooltipProvider>
        </ThemeProvider>
    </QueryClientProvider>
)

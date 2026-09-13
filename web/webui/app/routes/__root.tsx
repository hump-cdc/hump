import { createRootRoute, Outlet } from '@tanstack/react-router'

import type { FC } from 'react'

import { Providers } from '@/app'

const Root: FC = () => (
    <Providers>
        <Outlet />
    </Providers>
)

export const Route = createRootRoute({ component: Root })

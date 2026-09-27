import type { FC, PropsWithChildren } from 'react'

import { useConfig } from '../model/use-config'
import { ConfigError } from './config-error'
import { ConfigLoading } from './config-loading'

export const ConfigProvider: FC<PropsWithChildren> = ({ children }) => {
    const { isPending, isError, error } = useConfig()

    if (isPending) return <ConfigLoading />
    if (isError) return <ConfigError error={error} />

    return children
}

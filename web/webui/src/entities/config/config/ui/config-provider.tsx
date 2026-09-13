import type { FC, PropsWithChildren } from 'react'

import type { ErrorModelModel } from '@/shared/api'
import { useStorageInit } from '@/shared/storage'

import { useConfig } from '../model/use-config'
import { ConfigError } from './config-error'
import { ConfigLoading } from './config-loading'

const STORAGE_ERROR: ErrorModelModel = {
    status: 500,
    title: 'Local storage unavailable',
    detail: 'Failed to initialize the local operations database (IndexedDB).',
}

export const ConfigProvider: FC<PropsWithChildren> = ({ children }) => {
    const { isPending, isError, error, data } = useConfig()
    const { ready: storageReady, error: storageError } = useStorageInit(data)

    if (isPending) return <ConfigLoading />
    if (isError) return <ConfigError error={error} />
    if (storageError) return <ConfigError error={STORAGE_ERROR} />
    if (!storageReady) return <ConfigLoading />

    return children
}

import { useEffect, useState } from 'react'

import type { ConfigResponseBodyModel } from '@/shared/api'

import { resetAndInitStorage } from './init'

/** Readiness state for storage bootstrap, for gating the app behind a loader. */
export type StorageInitState = {
    ready: boolean
    error: unknown
}

/** Run storage bootstrap once `config` is available and report readiness. */
export const useStorageInit = (config: ConfigResponseBodyModel | undefined): StorageInitState => {
    const [state, setState] = useState<StorageInitState>({ ready: false, error: null })

    useEffect(() => {
        if (!config) return

        resetAndInitStorage(config).then(
            () => setState({ ready: true, error: null }),
            (error: unknown) => setState({ ready: false, error }),
        )
    }, [config])

    return state
}

import type { FC } from 'react'

import type { ErrorModelModel } from '@/shared/api'
import { ErrorLine, ErrorPrompt, ErrorTerminal } from '@/shared/ui/error'

type ConfigErrorProps = {
    error: ErrorModelModel
}

export const ConfigError: FC<ConfigErrorProps> = ({ error }) => {
    const status = error.status ?? 500
    const title = error.title ?? 'Something went wrong'

    return (
        <ErrorTerminal
            status={`${status} — configuration unavailable`}
            title="The cluster won't talk to us."
            description={`Error ${status}: ${title.toLowerCase()}.`}
        >
            <ErrorPrompt>hump serve</ErrorPrompt>
            <ErrorLine tone="danger">
                Error: failed to load configuration ({status} {title})
            </ErrorLine>

            <ErrorPrompt>cat /etc/hump/config.yaml</ErrorPrompt>
            <ErrorLine tone="muted">{error.detail || 'Failed to load the application configuration.'}</ErrorLine>

            <ErrorPrompt>systemctl restart hump</ErrorPrompt>
            <ErrorLine tone="warning">Job for hump.service failed.</ErrorLine>
        </ErrorTerminal>
    )
}

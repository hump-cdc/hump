import type { FC, ReactNode } from 'react'

import { SiApachekafka, SiClickhouse, SiNatsdotio, SiRabbitmq, SiSnowflake } from '@icons-pack/react-simple-icons'
import { Archive, Globe, Plug } from 'lucide-react'

import { cn } from '@/shared/utils'

/** Icon per known sink type (see internal/port/sink/sink_type.go) plus not-yet-supported ghost types. */
const SINK_ICONS: Record<string, ReactNode> = {
    kafka: <SiApachekafka />,
    'nats-core': <SiNatsdotio />,
    'nats-jetstream': <SiNatsdotio />,
    http: <Globe />,
    rabbitmq: <SiRabbitmq />,
    'rabbitmq-stream': <SiRabbitmq />,
    clickhouse: <SiClickhouse />,
    s3: <Archive />,
    snowflake: <SiSnowflake />,
}

/** Bare sink-type logo; unknown types fall back to a generic icon. */
export const SinkIcon: FC<{ type: string }> = ({ type }) => SINK_ICONS[type] ?? <Plug />

/** Square tile with the sink-type logo, used in sink node headers. */
export const SinkIconTile: FC<{ type: string; className?: string }> = ({ type, className }) => (
    <span
        className={cn(
            'flex size-7 shrink-0 items-center justify-center rounded-md border border-hairline bg-surface-2 text-primary [&_svg]:size-3.5',
            className,
        )}
    >
        <SinkIcon type={type} />
    </span>
)

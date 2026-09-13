import type { FC, HTMLAttributes, ReactNode, Ref } from 'react'

import { SiApachekafka, SiNatsdotio, SiRabbitmq } from '@icons-pack/react-simple-icons'
import { Globe, Plug } from 'lucide-react'

import { cn } from '@/shared/utils'

/** Friendly labels for the sink types defined in internal/types/sink/sink.go. */
const SINK_LABELS: Record<string, string> = {
    kafka: 'Kafka',
    'nats-core': 'NATS',
    'nats-jetstream': 'NATS JetStream',
    http: 'HTTP',
    rabbitmq: 'RabbitMQ',
    'rabbitmq-stream': 'RabbitMQ Stream',
}

/** Icon per known sink type; unknown types fall back to a generic icon. */
const SINK_ICONS: Record<string, ReactNode> = {
    kafka: <SiApachekafka />,
    'nats-core': <SiNatsdotio />,
    'nats-jetstream': <SiNatsdotio />,
    http: <Globe />,
    rabbitmq: <SiRabbitmq />,
    'rabbitmq-stream': <SiRabbitmq />,
}

type SinkProps = HTMLAttributes<HTMLDivElement> & {
    type: string
    host?: string
    ref?: Ref<HTMLDivElement>
}

export const Sink: FC<SinkProps> = ({ className, type, host, ...props }) => {
    const label = SINK_LABELS[type] ?? type
    const icon = SINK_ICONS[type] ?? <Plug />

    return (
        <div className={cn('flex items-center gap-1 text-sm', className)} {...props}>
            <span className="bg-surface-3 rounded-full p-1">
                <span className="flex shrink-0 items-center text-ink-subtle [&_svg]:size-3">{icon}</span>
            </span>
            <span className="text-ink">{label}</span>
            {host ? <span className="text-ink-muted/50">{host}</span> : null}
        </div>
    )
}

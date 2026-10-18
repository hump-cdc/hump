# Sink

An inline sink label: the sink type's icon, its friendly name and, optionally, the host. Use it
wherever a sink is named in a table cell, a list row or a card. Inside the flow diagrams the sink
logo comes from `SinkIcon` / `SinkIconTile` of `@/shared/ui/flow` instead.
Storybook: `Content/Sink`.

## Usage

```tsx
import { Sink } from '@/shared/ui/sink'

<Sink type="kafka" host="localhost:9092" />
<Sink type={sink.type} className="whitespace-nowrap" />
```

## Props

Plus every `div` attribute (`HTMLAttributes<HTMLDivElement>`).

| Prop | Type | Default | Description |
|---|---|---|---|
| `type` | `string` | required | sink type; picks the icon and the label |
| `host` | `string` | — | shown after the label; omitted when empty |
| `className` | `string` | — | merged onto the root `div` |
| `ref` | `Ref<HTMLDivElement>` | — | declared in the props type; reaches the `div` through the rest spread |

## Semantic styles

No variants. The root is `flex items-center gap-1 text-sm`.

| Part | Token | Signals |
|---|---|---|
| icon tile | `bg-surface-3`, round | a neutral chip behind the logo |
| icon | `text-ink-subtle`, `size-3` | secondary to the label |
| label | `text-ink` | the primary text |
| host | `text-ink-muted/50` | supporting detail |

`type` → label and icon:

| `type` | Label | Icon |
|---|---|---|
| `kafka` | Kafka | `SiApachekafka` |
| `nats-core` | NATS | `SiNatsdotio` |
| `nats-jetstream` | NATS JetStream | `SiNatsdotio` |
| `http` | HTTP | lucide `Globe` |
| `rabbitmq` | RabbitMQ | `SiRabbitmq` |
| `rabbitmq-stream` | RabbitMQ Stream | `SiRabbitmq` |
| anything else | the raw `type` string | lucide `Plug` |

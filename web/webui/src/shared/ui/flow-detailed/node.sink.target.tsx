import type { FC } from 'react'

import { Handle, Position } from '@xyflow/react'
import { cva } from 'class-variance-authority'

import { MetricHint } from '@/shared/ui/flow'
import { NumberFlow } from '@/shared/ui/number-flow'

import { sinkTargetKey, useTargetSelection } from './flow-detailed.context'
import type { SinkTarget } from './node.sink'

const rateVariants = cva('shrink-0 whitespace-nowrap rounded px-1.5 py-0.5 font-medium font-mono text-2xs', {
    variants: {
        health: {
            ok: 'bg-success-soft text-success',
            error: 'bg-danger-soft text-danger',
        },
    },
})

const targetRowVariants = cva(
    'flex h-7 w-full cursor-pointer items-center gap-2 rounded-md border bg-surface-3 px-2.5 text-left',
    {
        variants: {
            selected: {
                true: 'border-primary',
                false: 'border-hairline hover:border-hairline-strong',
            },
        },
        defaultVariants: {
            selected: false,
        },
    },
)

/**
 * One target row; owns the target handle so edges land on the row itself. The row is a
 * button: activating it selects the target instead of the sink card (the click doesn't
 * bubble to the node). The handle sits beside the button since a button can't contain it.
 */
export const TargetRow: FC<{ sinkId: string; target: SinkTarget }> = ({ sinkId, target }) => {
    const { selected, onTargetClick } = useTargetSelection()
    const isSelected = selected === sinkTargetKey(sinkId, target.id)

    return (
        <div className="relative">
            <button
                type="button"
                className={targetRowVariants({ selected: isSelected })}
                onClick={(event) => {
                    event.stopPropagation()
                    onTargetClick(sinkId, target)
                }}
            >
                <span className="min-w-0 flex-1 truncate font-medium font-mono text-ink text-sm">{target.name}</span>
                {target.count > 0 ? (
                    <MetricHint hint="Messages dispatched to this target, successful and failed">
                        <span className="shrink-0 whitespace-nowrap font-mono text-ink-subtle text-xs">
                            <NumberFlow value={target.count} kind="compact" />
                        </span>
                    </MetricHint>
                ) : null}
                {target.p95 !== undefined ? (
                    <MetricHint hint="p95 time to write one message to this target, retries included">
                        <span className="shrink-0 whitespace-nowrap rounded border border-hairline-strong bg-surface-2 px-1.5 font-mono text-ink-subtle text-xs">
                            <NumberFlow value={target.p95} kind="latency" />
                        </span>
                    </MetricHint>
                ) : null}
                {target.successRate !== undefined ? (
                    <MetricHint hint="Delivery success rate">
                        <span className={rateVariants({ health: target.health })}>
                            <NumberFlow value={target.successRate} kind="percent" />
                        </span>
                    </MetricHint>
                ) : null}
            </button>

            <Handle
                type="target"
                id={target.id}
                position={Position.Left}
                className={isSelected ? 'selected-target' : undefined}
            />
        </div>
    )
}

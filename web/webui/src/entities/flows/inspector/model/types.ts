/** A sink endpoint as the metrics identify it: its type and connection host. */
export type SinkKey = {
    type: string
    host: string
}

/**
 * The topology item the inspector is currently bound to, discriminated by `kind`. It carries only
 * identity and labels: the figures are resolved from the latest metrics, so they stay current.
 * `id` matches the canvas selection (for a target, `sinkTargetKey(sinkId, target)`).
 */
export type SelectedItem =
    | { kind: 'table'; id: string; name: string }
    | ({ kind: 'sink'; id: string; name: string } & SinkKey)
    | ({ kind: 'target'; id: string; name: string; sinkLabel: string } & SinkKey)
    /** `target` is set only for a detailed edge, which lands on one sink target. */
    | ({ kind: 'edge'; id: string; sourceLabel: string; targetLabel: string; table: string; target?: string } & SinkKey)

/** Selection payload of one kind, without the discriminant the store adds. */
export type SelectionOf<K extends SelectedItem['kind']> = Omit<Extract<SelectedItem, { kind: K }>, 'kind'>

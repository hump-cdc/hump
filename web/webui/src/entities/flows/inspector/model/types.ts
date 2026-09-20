import type { SinkNodeData as DetailedSinkNodeData, SinkTarget } from '@/shared/ui/flow-detailed'
import type { SinkNodeData, TableNodeData } from '@/shared/ui/flow-summary'

/**
 * The topology item the inspector is currently bound to. Discriminated by `kind`:
 * table/sink carry their rendered node data (`sink` from the summary view, `detailed-sink`
 * and `target` from the detailed one); edge and target carry pre-resolved labels (node
 * names) so the drawer needn't reach into the topology mock.
 */
export type SelectedItem =
    | { kind: 'table'; id: string; data: TableNodeData }
    | { kind: 'sink'; id: string; data: SinkNodeData }
    | { kind: 'detailed-sink'; id: string; data: DetailedSinkNodeData }
    /** `id` is `sinkTargetKey(sinkId, target.id)`, matching the detailed canvas selection. */
    | { kind: 'target'; id: string; sinkLabel: string; data: SinkTarget }
    | { kind: 'edge'; id: string; sourceLabel: string; targetLabel: string }

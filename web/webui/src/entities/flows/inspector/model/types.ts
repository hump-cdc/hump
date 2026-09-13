import type { SinkNodeData, TableNodeData } from '@/shared/ui/flow'

/**
 * The topology item the inspector is currently bound to. Discriminated by `kind`:
 * table/sink carry their rendered node data; edge carries pre-resolved endpoint
 * labels (node names) so the drawer needn't reach into the topology mock.
 */
export type SelectedItem =
    | { kind: 'table'; id: string; data: TableNodeData }
    | { kind: 'sink'; id: string; data: SinkNodeData }
    | { kind: 'edge'; id: string; sourceLabel: string; targetLabel: string }

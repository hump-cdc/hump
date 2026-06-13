import type { ComponentProps } from 'react'

import type { Edge } from '@xyflow/react'

import type { SinkNode, TableNode } from '@/shared/ui/flow'

/** Declarative `<TableNode>` config (id + table node data). */
export type TableSpec = ComponentProps<typeof TableNode>
/** Declarative `<SinkNode>` config (id + sink node data). */
export type SinkSpec = ComponentProps<typeof SinkNode>
/** A weighted connection between a table (source) and a sink (target). */
export type EdgeSpec = Edge

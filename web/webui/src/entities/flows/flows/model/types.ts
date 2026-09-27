import type { ComponentProps } from 'react'

import type { Edge } from '@xyflow/react'

import type { FlowDetailedSink } from '@/shared/ui/flow-detailed'
import type { FlowSummarySink, FlowSummaryTable } from '@/shared/ui/flow-summary'

/** Declarative `<FlowSummaryTable>` config (id + table node data); `<FlowDetailedTable>` shares the shape. */
export type TableSpec = ComponentProps<typeof FlowSummaryTable>
/** Declarative `<FlowSummarySink>` config (id + sink node data). */
export type SinkSpec = ComponentProps<typeof FlowSummarySink>
/** Declarative `<FlowDetailedSink>` config (id + sink node data with its targets). */
export type DetailedSinkSpec = ComponentProps<typeof FlowDetailedSink>
/** A weighted connection between a table (source) and a sink (target), optionally into a sink target (`targetHandle`). */
export type EdgeSpec = Edge

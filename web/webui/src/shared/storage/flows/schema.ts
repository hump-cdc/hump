import { db } from '../db'

/** Every store the flow graph spans, in the order transactions declare them. */
export const FLOW_STORES = () => [
    db.flowTables,
    db.flowSinks,
    db.flowSinkTargets,
    db.flowConnectionTableSink,
    db.flowConnectionTableTarget,
]

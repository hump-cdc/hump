/**
 * Delivery stats read from the scrape. `operationCount` is absolute; `p95`/`successRate` are
 * `undefined` when the window observed nothing, leaving the stored value in place.
 */
export type FlowDeliveryStats = {
    operationCount: number
    p95: number | undefined
    successRate: number | undefined
}

/**
 * Per-table CDC counters, one row per source table. The autoincrement `id` is the primary key
 * (absent only when writing a new row); `[schema+table]` is the unique natural key for upserts.
 */
export type FlowTableDto = {
    id?: number
    schema: string
    table: string
    insertCount: number
    updateCount: number
    deleteCount: number
    truncateCount: number
}

/** Per-sink delivery stats, one row per sink endpoint; `[sinkType+host]` is the unique natural key. */
export type FlowSinkDto = {
    id?: number
    sinkType: string
    host: string
    operationCount: number
    /** 95th-percentile delivery duration, in milliseconds. */
    p95: number
    /** Share of successful deliveries, 0–100. */
    successRate: number
}

/** Per-target stats within a sink (e.g. a topic or collection); `[sinkId+target]` is the unique natural key. */
export type FlowSinkTargetDto = {
    id?: number
    /** `FlowSinkDto.id` of the owning sink. */
    sinkId: number
    target: string
    operationCount: number
    /** 95th-percentile delivery duration, in milliseconds. */
    p95: number
    /** Share of successful deliveries, 0–100. */
    successRate: number
}

/** Table → sink edge counting the operations that flowed along it; `[tableId+sinkId]` is the unique natural key. */
export type FlowTableSinkDto = {
    id?: number
    /** `FlowTableDto.id` of the source table. */
    tableId: number
    /** `FlowSinkDto.id` of the destination sink. */
    sinkId: number
    operationCount: number
}

/** Table → sink-target edge counting the operations that flowed along it; `[tableId+targetId]` is the unique natural key. */
export type FlowTableTargetDto = {
    id?: number
    /** `FlowTableDto.id` of the source table. */
    tableId: number
    /** `FlowSinkTargetDto.id` of the destination target. */
    targetId: number
    operationCount: number
}

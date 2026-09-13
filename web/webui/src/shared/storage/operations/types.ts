import type { EventModel, RetryEventModel } from '@/shared/api'

/**
 * A persisted CDC operation, stored exactly as the `/api/operations` stream delivers it, with
 * retry attempts merged inline.
 */
export type OperationDto = EventModel & {
    retries?: RetryEventModel[]
}

/**
 * Selected filter facets for the operations feed. Empty facet lists match nothing; an empty
 * `search` adds no constraint.
 */
export type OperationsFilterCriteria = {
    /** Selected sink types. */
    sinks: string[]
    /** Selected tables, formatted as `schema.table`. */
    tables: string[]
    /** Selected operation kinds. */
    operations: string[]
    /** Selected delivery statuses. */
    statuses: string[]
    /** Free-text search; space-separated words are ANDed (substring, case-insensitive). */
    search: string
}

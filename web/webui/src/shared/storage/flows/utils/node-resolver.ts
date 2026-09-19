import { db } from '@/shared/storage'

/** Success rate a sink starts at before any delivery has been observed. */
const INITIAL_SUCCESS_RATE = 100

/**
 * Resolve node ids within one write pass, memoizing lookups for its lifetime since a batch
 * commonly touches the same table/sink/target repeatedly.
 */
export class NodeResolver {
    readonly #tables = new Map<string, number>()
    readonly #sinks = new Map<string, number>()
    readonly #targets = new Map<string, number>()

    /** `flow_tables` id for `[schema+table]`, inserting a zeroed row when absent. */
    async table(schema: string, table: string): Promise<number> {
        return this.#resolve(this.#tables, `${schema} ${table}`, () =>
            db.flowTables
                .where('[schema+table]')
                .equals([schema, table])
                .first()
                .then(
                    (existing) =>
                        existing?.id ??
                        db.flowTables.add({
                            schema,
                            table,
                            insertCount: 0,
                            updateCount: 0,
                            deleteCount: 0,
                            truncateCount: 0,
                        }),
                ),
        )
    }

    /** `flow_sinks` id for `[sinkType+host]`, inserting an idle row when absent. */
    async sink(sinkType: string, host: string): Promise<number> {
        return this.#resolve(this.#sinks, `${sinkType} ${host}`, () =>
            db.flowSinks
                .where('[sinkType+host]')
                .equals([sinkType, host])
                .first()
                .then(
                    (existing) =>
                        existing?.id ??
                        db.flowSinks.add({
                            sinkType,
                            host,
                            operationCount: 0,
                            p95: 0,
                            successRate: INITIAL_SUCCESS_RATE,
                        }),
                ),
        )
    }

    /** `flow_sink_targets` id for `[sinkId+target]`, inserting an idle row when absent. */
    async target(sinkId: number, target: string): Promise<number> {
        return this.#resolve(this.#targets, `${sinkId} ${target}`, () =>
            db.flowSinkTargets
                .where('[sinkId+target]')
                .equals([sinkId, target])
                .first()
                .then(
                    (existing) =>
                        existing?.id ??
                        db.flowSinkTargets.add({
                            sinkId,
                            target,
                            operationCount: 0,
                            p95: 0,
                            successRate: INITIAL_SUCCESS_RATE,
                        }),
                ),
        )
    }

    async #resolve(cache: Map<string, number>, key: string, lookup: () => Promise<number>): Promise<number> {
        const cached = cache.get(key)
        if (cached !== undefined) return cached

        const id = await lookup()
        cache.set(key, id)
        return id
    }
}

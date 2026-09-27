/** Rows per `/api/operations` request, and so per cached query. */
export const BLOCK_SIZE = 100

/**
 * Blocks covering rows `start..end` (inclusive), always including block 0 so the
 * total stays known wherever the viewport is. Sorted and deduplicated.
 */
export const blocksForRange = (start: number, end: number, blockSize: number): number[] => {
    const first = Math.max(0, Math.floor(start / blockSize))
    const last = Math.max(first, Math.floor(end / blockSize))
    const blocks = Array.from({ length: last - first + 1 }, (_, offset) => first + offset)

    return first === 0 ? blocks : [0, ...blocks]
}

/** Drops the blocks past the last one `total` rows fill; block 0 always stays. */
export const clampBlocks = (blocks: readonly number[], total: number, blockSize: number): number[] => {
    const last = Math.max(0, Math.ceil(total / blockSize) - 1)
    return blocks.filter((block) => block <= last)
}

/** Whether two block lists hold the same blocks in the same order. */
export const sameBlocks = (a: readonly number[], b: readonly number[]): boolean =>
    a.length === b.length && a.every((block, index) => block === b[index])

/** A loaded block: where it starts, and the total it was cut from. */
export type BlockPage<R> = {
    from: number
    total: number
    rows: readonly R[]
    /** When the page was fetched, in epoch ms; the newest page holds the freshest total. */
    updatedAt: number
}

/** `total` of the most recently fetched page, or `undefined` before any arrived. */
export const pickTotal = <R>(pages: readonly (BlockPage<R> | undefined)[]): number | undefined => {
    let newest: BlockPage<R> | undefined
    for (const page of pages) {
        if (page && (!newest || page.updatedAt > newest.updatedAt)) {
            newest = page
        }
    }

    return newest?.total
}

/** Row at `index` among the loaded pages, or `undefined` while its block is not loaded. */
export const rowAt = <R>(pages: ReadonlyMap<number, BlockPage<R>>, index: number, blockSize: number): R | undefined => {
    const block = Math.floor(index / blockSize)
    const page = pages.get(block)

    return page && page.from === block * blockSize ? page.rows[index - page.from] : undefined
}

import type { ColumnSizing } from './data-table.sizing'

const STORAGE_PREFIX = 'hump:data-table'

/**
 * Restores the persisted column widths of table `id`, keeping only the ids in
 * `validIds`. Also reads the array format the AG Grid version stored, taking the
 * entries that carried a width.
 */
export const loadColumnSizing = (id: string, validIds: readonly string[]): ColumnSizing => {
    try {
        const raw = localStorage.getItem(columnStateKey(id))
        return raw ? parseColumnSizing(JSON.parse(raw), validIds) : {}
    } catch {
        return {}
    }
}

/** Persists the column widths of table `id`; failures leave the current layout unaffected. */
export const saveColumnSizing = (id: string, sizing: ColumnSizing): void => {
    try {
        localStorage.setItem(columnStateKey(id), JSON.stringify(sizing))
    } catch {
        // Persistence is best-effort.
    }
}

/** Normalizes a stored value of either format to widths of known columns. */
export const parseColumnSizing = (value: unknown, validIds: readonly string[]): ColumnSizing => {
    const entries: [string, unknown][] = Array.isArray(value)
        ? value.flatMap((item: { colId?: unknown; width?: unknown; flex?: unknown }) =>
              typeof item?.colId === 'string' && item.flex == null ? [[item.colId, item.width]] : [],
          )
        : typeof value === 'object' && value !== null
          ? Object.entries(value)
          : []

    return Object.fromEntries(
        entries.filter(
            (entry): entry is [string, number] =>
                validIds.includes(entry[0]) && typeof entry[1] === 'number' && Number.isFinite(entry[1]),
        ),
    )
}

/** localStorage key of table `id`'s column widths. */
const columnStateKey = (id: string): string => `${STORAGE_PREFIX}:${id}:columns`

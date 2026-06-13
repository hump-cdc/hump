const STORAGE_PREFIX = 'hump:data-table'

const columnStateKey = (id: string): string => `${STORAGE_PREFIX}:${id}:columns`

export type ColumnLocalState = {
    colId: string
    width?: number | null
    flex?: number | null
}

export const loadColumnState = (id: string): ColumnLocalState[] | null => {
    try {
        const raw = localStorage.getItem(columnStateKey(id))
        if (!raw) {
            return null
        }
        const parsed = JSON.parse(raw)
        return Array.isArray(parsed) ? (parsed as ColumnLocalState[]) : null
    } catch {
        return null
    }
}

export const saveColumnState = (id: string, state: ColumnLocalState[]): void => {
    try {
        localStorage.setItem(columnStateKey(id), JSON.stringify(state))
    } catch {
        // Ignore persistence failures — the current session's layout is unaffected.
    }
}

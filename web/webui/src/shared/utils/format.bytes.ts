const decimal2 = new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

/** `undefined` in, `undefined` out; any number in, `string` out. */
type Formatted<T extends number | undefined> = T extends undefined ? undefined : string

/** Non-breaking space. */
const GROUP_SEPARATOR = ' '

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const

/** Byte size in 1024-based units, e.g. 512 → "512 B", 1536 → "1.50 KB", 5368709120 → "5.00 GB". */
export const formatBytes = <T extends number | undefined>(value: T): Formatted<T> => {
    if (value === undefined) return undefined as Formatted<T>

    let size = Math.abs(value)
    let unit = 0
    while (size >= 1024 && unit < UNITS.length - 1) {
        size /= 1024
        unit++
    }

    const sign = value < 0 ? '-' : ''
    const number = unit === 0 ? String(Math.round(size)) : decimal2.format(size).replaceAll(',', GROUP_SEPARATOR)

    return `${sign}${number} ${UNITS[unit]}` as Formatted<T>
}

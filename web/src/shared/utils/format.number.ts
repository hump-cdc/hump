const compact = new Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 2 })
const decimal = new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 })
const decimal1 = new Intl.NumberFormat('en-US', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
const decimal2 = new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

/** `undefined` in, `undefined` out; any number in, `string` out. */
type Formatted<T extends number | undefined> = T extends undefined ? undefined : string

/** Non-breaking space. */
const GROUP_SEPARATOR = ' '

/** Grouped integer, e.g. 1284300 → "1 284 300", (1284300.456, 'ms') → "1 284 300ms". */
export const formatNumber = <T extends number | undefined>(value: T, suffix?: string): Formatted<T> =>
    (value === undefined
        ? undefined
        : `${decimal.format(value).replaceAll(',', GROUP_SEPARATOR)}${suffix ?? ''}`) as Formatted<T>

/** Grouped number with 1 fraction digit, e.g. 1284300.456 → "1 284 300.5", (12.5, '%') → "12.5%". */
export const formatNumber1 = <T extends number | undefined>(value: T, suffix?: string): Formatted<T> =>
    (value === undefined
        ? undefined
        : `${decimal1.format(value).replaceAll(',', GROUP_SEPARATOR)}${suffix ?? ''}`) as Formatted<T>

/** Grouped number with 2 fraction digits, e.g. 1284300.456 → "1 284 300.46", (12.5, '%') → "12.50%". */
export const formatNumber2 = <T extends number | undefined>(value: T, suffix?: string): Formatted<T> =>
    (value === undefined
        ? undefined
        : `${decimal2.format(value).replaceAll(',', GROUP_SEPARATOR)}${suffix ?? ''}`) as Formatted<T>

/** Short magnitude, e.g. 3044300 → "3.04M". Past "T" it falls back to grouping, e.g. 1e18 → "1 000 000T". */
export const formatNumberCompact = <T extends number | undefined>(value: T): Formatted<T> =>
    (value === undefined ? undefined : compact.format(value).replaceAll(',', GROUP_SEPARATOR)) as Formatted<T>

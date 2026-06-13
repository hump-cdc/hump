const compact = new Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 2 })

/** Thousands-separated integer, e.g. 1284300 → "1,284,300". */
export const formatNumber = (value: number): string => Math.round(value).toLocaleString('en-US')

/** Short magnitude, e.g. 3044300 → "3.04M". */
export const formatNumberCompact = (value: number): string => compact.format(value)

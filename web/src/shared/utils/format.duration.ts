/** Seconds as "Mm Ss", e.g. 108 → "1m 48s". */
export const formatDuration = (seconds: number): string => {
    const total = Math.max(0, Math.round(seconds))
    const minutes = Math.floor(total / 60)
    const rest = total % 60
    return minutes > 0 ? `${minutes}m ${rest}s` : `${rest}s`
}

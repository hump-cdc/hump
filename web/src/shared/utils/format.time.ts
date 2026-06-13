const timeFormatter = new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3,
    hour12: false,
    timeZone: 'UTC',
})

/** Formats a `Date` as `HH:mm:ss.SSS` (UTC). */
export const formatTime = (d: Date): string => timeFormatter.format(d)

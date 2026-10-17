const timeFormatter = new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3,
    hour12: false,
})

/** Formats a `Date` as local `HH:mm:ss.SSS`. */
export const formatTime = (d: Date): string => timeFormatter.format(d)

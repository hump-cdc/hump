const dateTimeFormatter = new Intl.DateTimeFormat('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
})

/** Formats a `Date` as `DD Mon YYYY, HH:mm:ss`. */
export const formatDateTime = (d: Date): string => dateTimeFormatter.format(d)

const dateTimeFormatter = new Intl.DateTimeFormat('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
})

const timeOfDayFormatter = new Intl.DateTimeFormat('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
})

/** Formats a `Date` as `DD Mon YYYY, HH:mm:ss`. */
export const formatDateTime = (d: Date): string => dateTimeFormatter.format(d)

/** Formats a range as `DD Mon YYYY, HH:mm:ss → HH:mm:ss`, repeating the date only when the end falls on another day. */
export const formatDateTimeRange = (start: Date, end: Date): string => {
    const sameDay = start.toDateString() === end.toDateString()
    return `${formatDateTime(start)} → ${sameDay ? timeOfDayFormatter.format(end) : formatDateTime(end)}`
}

/** Formats a metric bucket timestamp as a zero-padded local `HH:MM` chart label. */
export const formatTimeCompact = (timestamp: number): string => {
    const date = new Date(timestamp)
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    return `${hours}:${minutes}`
}

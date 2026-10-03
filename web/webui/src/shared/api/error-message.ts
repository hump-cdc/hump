import type { ErrorModelModel } from './generated'

/** What an API call fails with: the `ErrorModel` body `throwOnError` throws, or a thrown `Error`. */
export type ApiError = Error | ErrorModelModel

declare module '@tanstack/react-query' {
    interface Register {
        defaultError: ApiError
    }
}

/** Readable message of an API failure. */
export const errorMessage = (error: ApiError): string => {
    if (error instanceof Error) return error.message
    if (isErrorModel(error)) return error.detail || error.title || String(error.status ?? 'Unknown error')

    return String(error)
}

/** Whether the value is an `application/problem+json` body; the static type alone cannot tell, as the client throws it untyped. */
const isErrorModel = (value: unknown): value is ErrorModelModel =>
    typeof value === 'object' &&
    value !== null &&
    ['detail', 'title', 'status'].some((key) => key in value) &&
    (!('detail' in value) || typeof value.detail === 'string') &&
    (!('title' in value) || typeof value.title === 'string')

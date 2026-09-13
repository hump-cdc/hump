import type { OperationDto } from './types'

/** Derived delivery status of an operation. */
export type OperationStatus = 'delivered' | 'failed'

/** Final delivery status: the outcome of the last attempt (newest retry, else the event itself). */
export const deriveOperationStatus = (record: OperationDto): OperationStatus => {
    const last = record.retries?.[record.retries.length - 1]
    const success = last ? last.success : record.success
    return success ? 'delivered' : 'failed'
}

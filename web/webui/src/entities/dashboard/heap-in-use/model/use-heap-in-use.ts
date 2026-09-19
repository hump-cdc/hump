import { useLiveControlsChartStep } from '@/entities/config/state/@x/dashboard'
import { useMetrics } from '@/entities/metrics/@x/dashboard'
import { formatTimeCompact } from '@/shared/utils'

/** A single point on the heap area chart, in megabytes. */
type Item = { time: string; heapInUse: number }

/** Convert a byte reading to megabytes, rounded to 2 fraction digits. */
const toMegabytes = (bytes: number): number => Math.round((bytes / 1024 / 1024) * 100) / 100

/** Heap chart at the chart step from the live controls. `value` is the latest reading in megabytes. */
export const useHeapInUse = (): { value: number | undefined; values: Item[] } => {
    const chartStep = useLiveControlsChartStep()
    const { data } = useMetrics(chartStep)
    const points = data?.points

    // Points run oldest → newest, so the last one is the newest bucket the response reported.
    const latest = points?.at(-1)?.memoryAllocBytes
    const value = latest === undefined ? undefined : toMegabytes(latest)
    const values: Item[] = (points ?? []).map(({ timestamp, memoryAllocBytes }) => ({
        time: formatTimeCompact(timestamp),
        heapInUse: toMegabytes(memoryAllocBytes),
    }))

    return { value, values }
}

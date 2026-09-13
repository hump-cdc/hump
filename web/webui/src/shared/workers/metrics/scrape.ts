import { Metrics } from '@/shared/api'
import { createParser, type PromMetrics, TRACKED_METRICS } from '@/shared/prometheus'

// One parser for the process: it holds the length index the tracked-family filter runs on.
const parse = createParser(TRACKED_METRICS)

/** Fetch one scrape and parse it. `undefined` when the response carried no body. */
export const scrape = async (signal: AbortSignal): Promise<PromMetrics | undefined> => {
    const { data } = await Metrics.get({ signal, responseTransformer: undefined })
    if (!data) return undefined

    return parse(data)
}

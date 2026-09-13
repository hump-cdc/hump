import { metricsWorker } from './metrics'
import { operationsStreamWorker } from './operations-stream'

// Both feeds of the live dashboard, started and stopped together: the metrics poll corrects
// what the operations stream counted, so running one alone skews the flow graph.
export const WORKERS = [metricsWorker, operationsStreamWorker]

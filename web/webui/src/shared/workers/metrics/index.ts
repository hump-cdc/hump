import { WorkerManager } from '@/shared/workers/manager'

export const metricsWorker = new WorkerManager(
    () => new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' }),
)

import { WorkerManager } from '@/shared/workers/manager'

export const operationsStreamWorker = new WorkerManager(
    () => new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' }),
)

export type Attempt = {
    attempt: number
    started: Date
    duration: string
    result: 'failed' | 'delivered'
    error?: string
}

export type Operation = {
    id: string
    time: Date
    table: string
    op: 'insert' | 'update' | 'delete' | 'select'
    sink: string
    status: 'delivered' | 'failed' | 'retrying'
    attempts: number
    duration: string
    error?: string
    retries?: Attempt[]
}

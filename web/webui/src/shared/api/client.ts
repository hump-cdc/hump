import type { CreateClientConfig } from './generated/client.gen'

const baseUrl = (typeof API_URL !== 'undefined' && API_URL) || globalThis.location?.origin || ''

export const createClientConfig: CreateClientConfig = (config) => ({
    ...config,
    baseUrl,
})

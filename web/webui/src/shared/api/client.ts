import type { CreateClientConfig } from './generated/client.gen'

const baseUrl = API_URL || globalThis.location.origin

export const createClientConfig: CreateClientConfig = (config) => ({
    ...config,
    baseUrl,
})

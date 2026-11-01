import { client } from './generated/client.gen'

export * from './error-message'
export * from './generated'
export * from './generated/@tanstack/react-query.gen'
export * from './generated/zod.gen'

// The generator strips Content-Type from bodyless requests; stopping still requires JSON for CSRF protection.
client.interceptors.request.use((request, options): Request => {
    if (options.method === 'POST' && options.url === '/api/snapshot/stop') {
        request.headers.set('Content-Type', 'application/json')
    }
    return request
})

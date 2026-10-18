import { describe, expect, test } from 'bun:test'

import { type ApiError, errorMessage } from './error-message'

describe('errorMessage', () => {
    test('reads the message of a thrown Error', () => {
        expect(errorMessage(new Error('connection refused'))).toBe('connection refused')
    })

    test('prefers the detail of a problem body, then its title, then its status', () => {
        expect(errorMessage({ detail: 'table not found', title: 'Not Found', status: 404 })).toBe('table not found')
        expect(errorMessage({ title: 'Not Found', status: 404 })).toBe('Not Found')
        expect(errorMessage({ status: 404 })).toBe('404')
    })

    test('skips an empty detail', () => {
        expect(errorMessage({ detail: '', title: 'Bad Request' })).toBe('Bad Request')
    })

    test('falls back to a generic message when the problem body has no readable field', () => {
        expect(errorMessage({ detail: '', title: '' })).toBe('Unknown error')
    })

    test('stringifies a value that is not a problem body', () => {
        // The client throws the body untyped, so `ApiError` does not rule these out at runtime.
        expect(errorMessage('timeout' as unknown as ApiError)).toBe('timeout')
        expect(errorMessage({ detail: 42 } as unknown as ApiError)).toBe('[object Object]')
        expect(errorMessage({ code: 'E1' } as unknown as ApiError)).toBe('[object Object]')
    })
})

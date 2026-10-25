import { describe, expect, test } from 'bun:test'

import { cn } from './cn'

describe('cn', () => {
    test('joins conditional classes and lets the later Tailwind class win', () => {
        expect(cn('p-2', false, { 'text-sm': true, hidden: false }, 'p-4')).toBe('text-sm p-4')
    })

    test('treats the custom glow shadow as a shadow utility', () => {
        expect(cn('shadow-sm', 'shadow-glow')).toBe('shadow-glow')
    })
})

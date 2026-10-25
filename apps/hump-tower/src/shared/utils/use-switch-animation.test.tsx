import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { act, cleanup, renderHook } from '@testing-library/react'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'

import { SwitchViewValueContext, useSwitchAnimation, useSwitchViewValue } from './use-switch-animation'

beforeAll(registerDom)
afterAll(unregisterDom)
afterEach(cleanup)

describe('useSwitchAnimation', () => {
    test('renders one view until the value changes', () => {
        const { result } = renderHook(() => useSwitchAnimation('a'))

        expect(result.current).toMatchObject({ current: 'a', leaving: null, views: ['a'] })
    })

    test('keeps the previous value as the leaving view until its exit ends', () => {
        const { result, rerender } = renderHook(({ value }: { value: string }) => useSwitchAnimation(value), {
            initialProps: { value: 'a' },
        })

        rerender({ value: 'b' })
        expect(result.current).toMatchObject({ current: 'b', leaving: 'a', views: ['a', 'b'] })

        rerender({ value: 'c' })
        expect(result.current.views).toEqual(['b', 'c'])

        act(() => result.current.onLeaveEnd())
        expect(result.current).toMatchObject({ current: 'c', leaving: null, views: ['c'] })
    })
})

describe('useSwitchViewValue', () => {
    test('reads the value of the enclosing view', () => {
        const { result } = renderHook(() => useSwitchViewValue<string>(), {
            wrapper: ({ children }) => (
                <SwitchViewValueContext.Provider value="view">{children}</SwitchViewValueContext.Provider>
            ),
        })

        expect(result.current).toBe('view')
    })

    test('is undefined outside a view', () => {
        const { result } = renderHook(() => useSwitchViewValue<string>())

        expect(result.current).toBeUndefined()
    })
})

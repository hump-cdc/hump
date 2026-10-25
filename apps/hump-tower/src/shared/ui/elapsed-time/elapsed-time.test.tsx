import { afterAll, afterEach, beforeAll, describe, expect, spyOn, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, render } from '@testing-library/react'

import { ElapsedTime } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('ElapsedTime', () => {
    test('ticks every second while running and stops on unmount', () => {
        const setIntervalSpy = spyOn(globalThis, 'setInterval')
        const clearIntervalSpy = spyOn(globalThis, 'clearInterval')
        const { container, unmount } = render(<ElapsedTime startedAt={new Date(Date.now() - 5_000)} />)

        expect(setIntervalSpy).toHaveBeenCalledTimes(1)
        const [tick, delay] = setIntervalSpy.mock.calls[0] ?? []
        expect(delay).toBe(1_000)
        act(() => {
            if (typeof tick === 'function') tick()
        })
        expect(container.firstElementChild).not.toBeNull()

        unmount()
        expect(clearIntervalSpy).toHaveBeenCalled()
        setIntervalSpy.mockRestore()
        clearIntervalSpy.mockRestore()
    })

    test('shows a fixed duration without a timer once finished', () => {
        const setIntervalSpy = spyOn(globalThis, 'setInterval')
        const { container } = render(
            <ElapsedTime startedAt={new Date(0)} finishedAt={new Date(65_000)} className="text-sm" />,
        )

        expect(setIntervalSpy).not.toHaveBeenCalled()
        expect(container.firstElementChild).not.toBeNull()
        setIntervalSpy.mockRestore()
    })
})

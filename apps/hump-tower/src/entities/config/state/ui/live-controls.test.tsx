import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, type Mock, mock, spyOn, test } from 'bun:test'

import { act, cleanup, fireEvent, renderHook } from '@testing-library/react'

import { registerDom, unregisterDom } from '@/shared/testing/dom.fake'
import { fakeLiveStream } from '@/shared/testing/live.fake'
import { renderWithRouter } from '@/shared/testing/render.fake'

import { useLiveControlsChartStep as useFlowsChartStep } from '../@x/flows'
import { useLiveControlsChartStep as useInspectorChartStep } from '../@x/inspector'
import { useLiveControlsChartStep as useOperationsChartStep } from '../@x/operations'
import { useLiveControlsChartStep as useStatisticsChartStep } from '../@x/statistics'
import { LiveControls } from '../index'
import { useLiveControlsStore } from '../model/store'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))

beforeAll(registerDom)
afterAll(unregisterDom)

// The persist middleware warns when it cannot write; the warning is muted.
let warn: Mock<typeof console.warn>

beforeEach(() => {
    warn = spyOn(console, 'warn').mockImplementation(() => {})
    fakeLiveStream.reset()
    useLiveControlsStore.setState(useLiveControlsStore.getInitialState(), true)
})

afterEach(() => {
    cleanup()
    warn.mockRestore()
})

describe('LiveControls', () => {
    test('labels each stream status on the badge', async () => {
        const { getByText } = await renderWithRouter(<LiveControls />)
        expect(getByText('offline')).toBeDefined()

        act(() => fakeLiveStream.setStatus('connecting'))
        expect(getByText('connecting')).toBeDefined()

        act(() => fakeLiveStream.setStatus('live'))
        expect(getByText('live')).toBeDefined()

        act(() => fakeLiveStream.setStatus('reconnecting'))
        expect(getByText('offline')).toBeDefined()
    })

    test('links to the snapshot page only while a snapshot is processing', async () => {
        const { container, queryByText } = await renderWithRouter(<LiveControls />)
        expect(queryByText('Snapshot')).toBeNull()

        act(() => fakeLiveStream.setSnapshot({ state: 'processing', progress: 40 }))
        expect(container.querySelector('a[href="/snapshot"]')?.textContent).toContain('Snapshot')

        act(() => fakeLiveStream.setSnapshot({ state: 'done', progress: 100 }))
        expect(container.querySelector('a[href="/snapshot"]')).toBeNull()
    })

    test('switches the chart step from the dropdown', async () => {
        const { getByText, findByRole } = await renderWithRouter(<LiveControls />)

        fireEvent.keyDown(getByText('offline'), { key: 'Enter' })
        fireEvent.mouseDown(await findByRole('tab', { name: '5m' }), { button: 0 })

        expect(useLiveControlsStore.getState().chartStep).toBe('5m')
    })
})

describe('useLiveControlsChartStep', () => {
    test('reads the selected step through every cross-import', () => {
        useLiveControlsStore.getState().setChartStep('10m')

        for (const useChartStep of [
            useFlowsChartStep,
            useInspectorChartStep,
            useOperationsChartStep,
            useStatisticsChartStep,
        ]) {
            const { result } = renderHook(() => useChartStep())
            expect(result.current).toBe('10m')
        }
    })
})

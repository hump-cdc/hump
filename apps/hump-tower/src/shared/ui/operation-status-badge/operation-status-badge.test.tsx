import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { OperationStatusBadge } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('OperationStatusBadge', () => {
    test('compact variant shows a bare icon per outcome', () => {
        const failed = render(<OperationStatusBadge status="failed" attempts={1} variant="compact" className="x" />)
        expect(failed.getByLabelText('Failed').getAttribute('class')).toContain('text-danger')
        failed.unmount()

        const delivered = render(<OperationStatusBadge status="delivered" attempts={3} variant="compact" />)
        expect(delivered.getByLabelText('Delivered').getAttribute('class')).toContain('text-success')
    })

    test('labels a first-attempt outcome without the attempt count', () => {
        expect(render(<OperationStatusBadge status="failed" attempts={1} />).getByText('Failed')).toBeDefined()
        expect(render(<OperationStatusBadge status="delivered" attempts={1} />).getByText('Delivered')).toBeDefined()
    })

    test('mentions the retry for a retried outcome', () => {
        expect(
            render(<OperationStatusBadge status="failed" attempts={2} />).getByText('Failed after retry'),
        ).toBeDefined()
        expect(
            render(<OperationStatusBadge status="delivered" attempts={2} />).getByText('Delivered after retry'),
        ).toBeDefined()
    })

    test('spells the attempt count out with `withAttempts`', () => {
        const view = render(<OperationStatusBadge status="failed" attempts={4} withAttempts />)

        expect(view.getByText('Failed after 4 attempts')).toBeDefined()
    })
})

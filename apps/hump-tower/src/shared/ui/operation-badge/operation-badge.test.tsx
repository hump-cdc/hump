import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { OperationBadge } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('OperationBadge', () => {
    test.each(['insert', 'update', 'delete', 'truncate'] as const)('labels the %s operation', (op) => {
        const view = render(<OperationBadge op={op} className="custom" />)

        const badge = view.getByText(op)
        expect(badge.className).toContain('uppercase')
        expect(badge.className).toContain('custom')
    })

    test('colors each operation differently', () => {
        const classes = (['insert', 'update', 'delete', 'truncate'] as const).map((op) => {
            const view = render(<OperationBadge op={op} />)
            const className = view.getByText(op).className
            view.unmount()
            return className
        })

        expect(new Set(classes).size).toBe(4)
    })
})

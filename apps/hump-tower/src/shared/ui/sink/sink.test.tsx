import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { Sink } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('Sink', () => {
    test('labels a known sink type and shows its host', () => {
        const view = render(<Sink type="nats-jetstream" host="nats:4222" className="x" />)

        expect(view.getByText('NATS JetStream')).toBeDefined()
        expect(view.getByText('nats:4222')).toBeDefined()
        expect(view.container.querySelector('svg')).not.toBeNull()
        expect(view.container.firstElementChild?.className).toContain('x')
    })

    test('falls back to the raw type and a generic icon for an unknown sink', () => {
        const view = render(<Sink type="carrier-pigeon" />)

        expect(view.getByText('carrier-pigeon')).toBeDefined()
        expect(view.container.querySelector('svg')).not.toBeNull()
        expect(view.container.firstElementChild?.children).toHaveLength(2)
    })
})

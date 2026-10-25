import { afterAll, afterEach, beforeAll, describe, expect, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import { ErrorLine, ErrorPrompt, ErrorTerminal } from './index'

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('ErrorTerminal', () => {
    test('renders the terminal lines, status, texts and action', () => {
        const { getByRole, getByText } = render(
            <ErrorTerminal
                status="404"
                title="Lost"
                description="Not here"
                action={<button type="button">Home</button>}
            >
                <ErrorPrompt>cd /page</ErrorPrompt>
                <ErrorLine tone="danger">no such page</ErrorLine>
            </ErrorTerminal>,
        )

        expect(getByRole('heading').textContent).toBe('Lost')
        expect(getByText('Not here')).toBeDefined()
        expect(getByText('status: 404')).toBeDefined()
        expect(getByText('bash — hump — 80×24')).toBeDefined()
        expect(getByText('cd /page')).toBeDefined()
        expect(getByText('no such page').className).toContain('text-[#eb5757]')
        expect(getByRole('button').textContent).toBe('Home')
    })

    test('omits the action and uses a custom terminal title', () => {
        const { queryByRole, getByText } = render(
            <ErrorTerminal status="500" title="Down" description="Broken" terminalTitle="zsh" />,
        )

        expect(queryByRole('button')).toBeNull()
        expect(getByText('zsh')).toBeDefined()
    })
})

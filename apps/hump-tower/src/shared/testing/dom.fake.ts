import { GlobalRegistrator } from '@happy-dom/global-registrator'

import { fakeApi } from './api.fake'

/** Origin of the fake page; relative API URLs resolve against it. */
const ORIGIN = 'http://localhost/'

/**
 * Installs happy-dom on `ORIGIN` with the fake API in place of `fetch`; pair with `unregisterDom`.
 * Under bun, `@number-flow/react` takes its server build and never defines its element, while its
 * wrapper calls the element's update hooks whenever the value changes; a no-op element stands in.
 */
export const registerDom = (): void => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register({ url: ORIGIN })
    globalThis.fetch = Object.assign(fakeApi.fetch, { preconnect: () => {} })
    if (!customElements.get('number-flow-react')) customElements.define('number-flow-react', createNumberFlowStub())
}

/**
 * Removes happy-dom again, which restores the runtime's own `fetch`. React work still scheduled by
 * the last test (a router update, a settled query) runs first, while `window` still exists.
 */
export const unregisterDom = async (): Promise<void> => {
    await new Promise((resolve) => setTimeout(resolve, 10))
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
}

/** Stand-in for the `number-flow-react` element, ignoring updates; built once `HTMLElement` exists. */
const createNumberFlowStub = (): CustomElementConstructor =>
    class extends HTMLElement {
        willUpdate(): void {}
        didUpdate(): void {}
    }

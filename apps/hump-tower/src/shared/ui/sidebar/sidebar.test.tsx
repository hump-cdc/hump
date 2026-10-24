import { type FC, useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, mock, spyOn, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { act, cleanup, fireEvent, render } from '@testing-library/react'

import {
    SIDEBAR_STORAGE_KEY,
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarGroup,
    SidebarGroupContent,
    SidebarGroupLabel,
    SidebarHeader,
    SidebarInset,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarProvider,
    SidebarRail,
    SidebarSeparator,
    SidebarTrigger,
    useSidebar,
} from './index'

// Radix picks its layout effect when first imported, as a no-op without a DOM; bun shares modules
// across test files, so point it at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))

/** The happy-dom window, whose `matchMedia` follows the viewport set here. */
type HappyDomWindow = { happyDOM: { setViewport: (viewport: { width: number }) => void } }

/** Whether the global window is happy-dom's, i.e. carries its `happyDOM` controls. */
const isHappyDomWindow = (value: object): value is HappyDomWindow => 'happyDOM' in value

/** Resizes the happy-dom viewport, which `useIsMobile` reads through `matchMedia`. */
const setViewportWidth = (width: number): void => {
    if (!isHappyDomWindow(window)) throw new Error('happy-dom is not registered')
    window.happyDOM.setViewport({ width })
}

/** Makes every `localStorage` access throw until the returned function restores it. */
const blockLocalStorage = (): (() => void) => {
    const original = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
    Object.defineProperty(globalThis, 'localStorage', {
        configurable: true,
        get: () => {
            throw new Error('blocked')
        },
    })

    return () => {
        if (original) Object.defineProperty(globalThis, 'localStorage', original)
    }
}

/** Shows the sidebar context and drives `setOpen` with a plain boolean. */
const Probe: FC = () => {
    const { state, open, openMobile, isMobile, setOpen } = useSidebar()

    return (
        <>
            <output>{`${state} ${open} ${openMobile} ${isMobile}`}</output>
            <button type="button" onClick={() => setOpen(false)}>
                Collapse
            </button>
        </>
    )
}

/** A full sidebar layout with every part in place. */
const Layout: FC<{ side?: 'left' | 'right' }> = ({ side }) => (
    <>
        <Sidebar side={side} className="container-x">
            <SidebarHeader>Header</SidebarHeader>
            <SidebarContent>
                <SidebarGroup>
                    <SidebarGroupLabel>Pages</SidebarGroupLabel>
                    <SidebarGroupLabel asChild>
                        <h2>Slotted label</h2>
                    </SidebarGroupLabel>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            <SidebarMenuItem>
                                <SidebarMenuButton isActive size="lg">
                                    Plain
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                            <SidebarMenuItem>
                                <SidebarMenuButton tooltip="Flows">Flows</SidebarMenuButton>
                            </SidebarMenuItem>
                            <SidebarMenuItem>
                                <SidebarMenuButton asChild tooltip={{ children: 'Stats', className: 'tip' }}>
                                    <a href="/statistics">Statistics</a>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>
                <SidebarSeparator />
            </SidebarContent>
            <SidebarFooter>Footer</SidebarFooter>
        </Sidebar>
        <SidebarInset>
            <SidebarTrigger />
            <Probe />
        </SidebarInset>
    </>
)

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
beforeEach(() => localStorage.clear())
afterEach(() => {
    cleanup()
    setViewportWidth(1024)
})

describe('useSidebar', () => {
    test('throws outside a SidebarProvider', () => {
        const consoleError = spyOn(console, 'error').mockImplementation(() => {})

        expect(() => render(<SidebarRail />)).toThrow('useSidebar must be used within a SidebarProvider.')

        consoleError.mockRestore()
    })
})

describe('SidebarProvider', () => {
    test('renders the desktop sidebar with every part and toggles it', () => {
        const onClick = mock(() => {})
        const view = render(
            <SidebarProvider className="wrapper-x" style={{ color: 'red' }}>
                <Layout side="right" />
                <SidebarTrigger onClick={onClick}>Custom</SidebarTrigger>
            </SidebarProvider>,
        )
        const wrapper = view.container.querySelector<HTMLElement>('[data-slot="sidebar-wrapper"]')
        const sidebar = view.container.querySelector<HTMLElement>('[data-slot="sidebar"]')

        expect(wrapper?.className).toContain('wrapper-x')
        expect(wrapper?.style.getPropertyValue('--sidebar-width')).toBe('16rem')
        expect(wrapper?.style.color).toBe('red')
        expect(sidebar?.dataset.side).toBe('right')
        expect(sidebar?.dataset.state).toBe('expanded')
        expect(view.container.querySelector('[data-slot="sidebar-container"]')?.className).toContain('right-0')
        expect(view.getByText('Slotted label').tagName).toBe('H2')
        expect(view.getByText('Plain').dataset.active).toBe('true')
        expect(view.getByText('Statistics').tagName).toBe('A')
        for (const slot of [
            'header',
            'content',
            'group',
            'group-content',
            'menu',
            'menu-item',
            'separator',
            'footer',
        ]) {
            expect(view.container.querySelector(`[data-sidebar="${slot}"]`)).not.toBeNull()
        }
        expect(view.getByText('expanded true false false')).toBeDefined()

        fireEvent.click(view.getByText('Custom'))
        expect(onClick).toHaveBeenCalledTimes(1)
        expect(sidebar?.dataset.state).toBe('collapsed')
        expect(sidebar?.dataset.collapsible).toBe('icon')
        expect(localStorage.getItem(SIDEBAR_STORAGE_KEY)).toBe('false')

        fireEvent.click(view.getByTitle('Toggle sidebar'))
        expect(sidebar?.dataset.state).toBe('expanded')

        fireEvent.click(view.getByText('Collapse'))
        expect(sidebar?.dataset.state).toBe('collapsed')
    })

    test('toggles on Cmd/Ctrl+B only', () => {
        const view = render(
            <SidebarProvider>
                <Probe />
            </SidebarProvider>,
        )

        fireEvent.keyDown(window, { key: 'b' })
        expect(view.getByText('expanded true false false')).toBeDefined()

        fireEvent.keyDown(window, { key: 'b', ctrlKey: true })
        expect(view.getByText('collapsed false false false')).toBeDefined()

        fireEvent.keyDown(window, { key: 'b', metaKey: true })
        expect(view.getByText('expanded true false false')).toBeDefined()
    })

    test.each([
        ['true', false, 'expanded'],
        ['false', true, 'collapsed'],
        [undefined, false, 'collapsed'],
    ] as const)('starts from the stored preference %p over defaultOpen=%p', (stored, defaultOpen, state) => {
        if (stored !== undefined) localStorage.setItem(SIDEBAR_STORAGE_KEY, stored)
        const view = render(
            <SidebarProvider defaultOpen={defaultOpen}>
                <Probe />
            </SidebarProvider>,
        )

        expect(view.getByRole('status').textContent).toStartWith(state)
    })

    test('keeps working while localStorage is unavailable', () => {
        const restore = blockLocalStorage()
        try {
            const view = render(
                <SidebarProvider defaultOpen={false}>
                    <Probe />
                    <SidebarTrigger />
                </SidebarProvider>,
            )
            expect(view.getByText('collapsed false false false')).toBeDefined()

            fireEvent.click(view.getByRole('button', { name: 'Toggle sidebar' }))
            expect(view.getByText('expanded true false false')).toBeDefined()
        } finally {
            restore()
        }
    })

    test('reports changes without applying them while controlled', () => {
        const onOpenChange = mock((_open: boolean) => {})
        const view = render(
            <SidebarProvider open onOpenChange={onOpenChange}>
                <Probe />
                <SidebarTrigger />
            </SidebarProvider>,
        )

        fireEvent.click(view.getByRole('button', { name: 'Toggle sidebar' }))

        expect(onOpenChange).toHaveBeenCalledWith(false)
        expect(view.getByText('expanded true false false')).toBeDefined()
    })

    test('renders the sidebar as a sheet on mobile', async () => {
        setViewportWidth(500)
        const view = render(
            <SidebarProvider>
                <Layout />
            </SidebarProvider>,
        )

        expect(view.getByText('expanded true false true')).toBeDefined()
        expect(document.body.querySelector('[data-mobile="true"]')).toBeNull()

        await act(async () => fireEvent.click(view.getByRole('button', { name: 'Toggle sidebar' })))

        const sheet = document.body.querySelector<HTMLElement>('[data-mobile="true"]')
        expect(sheet?.style.getPropertyValue('--sidebar-width')).toBe('18rem')
        expect(sheet?.textContent).toContain('Flows')
        expect(view.getByText('expanded true true true')).toBeDefined()
    })
})

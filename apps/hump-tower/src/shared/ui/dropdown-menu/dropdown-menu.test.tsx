import { useLayoutEffect } from 'react'

import { afterAll, afterEach, beforeAll, describe, expect, mock, test } from 'bun:test'

import { GlobalRegistrator } from '@happy-dom/global-registrator'
import { cleanup, render } from '@testing-library/react'

import {
    DropdownMenu,
    DropdownMenuCheckboxItem,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuPortal,
    DropdownMenuRadioGroup,
    DropdownMenuRadioItem,
    DropdownMenuSeparator,
    DropdownMenuShortcut,
    DropdownMenuSub,
    DropdownMenuSubContent,
    DropdownMenuSubTrigger,
    DropdownMenuTrigger,
} from './index'

// Radix and Base UI pick their layout effect when first imported, as a no-op without a DOM; bun shares
// modules across test files, so point them at React's own one whatever order the files load in.
mock.module('@radix-ui/react-use-layout-effect', () => ({ useLayoutEffect }))
mock.module('@base-ui/utils/useIsoLayoutEffect', () => ({ useIsoLayoutEffect: useLayoutEffect }))

beforeAll(() => {
    if (!GlobalRegistrator.isRegistered) GlobalRegistrator.register()
})
afterAll(async () => {
    if (GlobalRegistrator.isRegistered) await GlobalRegistrator.unregister()
})
afterEach(cleanup)

describe('DropdownMenu', () => {
    test('renders every part of an open menu and its open submenu', async () => {
        const { findAllByRole } = render(
            <DropdownMenu open modal={false}>
                <DropdownMenuTrigger>Open</DropdownMenuTrigger>
                <DropdownMenuContent>
                    <DropdownMenuLabel inset>Columns</DropdownMenuLabel>
                    <DropdownMenuGroup>
                        <DropdownMenuItem variant="destructive">
                            Delete
                            <DropdownMenuShortcut>⌘D</DropdownMenuShortcut>
                        </DropdownMenuItem>
                        <DropdownMenuCheckboxItem checked>Name</DropdownMenuCheckboxItem>
                    </DropdownMenuGroup>
                    <DropdownMenuSeparator />
                    <DropdownMenuRadioGroup value="a">
                        <DropdownMenuRadioItem value="a">A</DropdownMenuRadioItem>
                    </DropdownMenuRadioGroup>
                    <DropdownMenuSub open>
                        <DropdownMenuSubTrigger inset>More</DropdownMenuSubTrigger>
                        <DropdownMenuPortal>
                            <DropdownMenuSubContent>
                                <DropdownMenuItem>Nested</DropdownMenuItem>
                            </DropdownMenuSubContent>
                        </DropdownMenuPortal>
                    </DropdownMenuSub>
                </DropdownMenuContent>
            </DropdownMenu>,
        )
        const [menu, submenu] = await findAllByRole('menu', { hidden: true })
        const slots = [...document.querySelectorAll('[data-slot]')].map((element) => element.getAttribute('data-slot'))

        expect(menu?.getAttribute('data-slot')).toBe('dropdown-menu-content')
        expect(submenu?.getAttribute('data-slot')).toBe('dropdown-menu-sub-content')
        expect(slots).toEqual(
            expect.arrayContaining([
                'dropdown-menu-trigger',
                'dropdown-menu-label',
                'dropdown-menu-group',
                'dropdown-menu-item',
                'dropdown-menu-shortcut',
                'dropdown-menu-checkbox-item',
                'dropdown-menu-separator',
                'dropdown-menu-radio-group',
                'dropdown-menu-radio-item',
                'dropdown-menu-sub-trigger',
                'dropdown-menu-sub-content',
            ]),
        )
    })
})

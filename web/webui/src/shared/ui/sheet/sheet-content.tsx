import type { ComponentProps, FC } from 'react'

import {
    Close as SheetClosePrimitive,
    Content as SheetContentPrimitive,
    Portal as SheetPortal,
} from '@radix-ui/react-dialog'
import { cva, type VariantProps } from 'class-variance-authority'
import { X } from 'lucide-react'

import { cn } from '@/shared/utils'

import { SheetOverlay } from './sheet-overlay'

const sheetContentVariants = cva(
    [
        'fixed z-50 flex flex-col gap-2 overflow-hidden border-hairline bg-canvas text-ink shadow-xl',
        'transition ease-in-out data-[state=closed]:animate-out data-[state=open]:animate-in data-[state=closed]:duration-200 data-[state=open]:duration-300',
    ],
    {
        variants: {
            side: {
                right: 'data-[state=open]:slide-in-from-right data-[state=closed]:slide-out-to-right inset-y-0 right-0 h-full w-3/4 border-l sm:max-w-sm',
                left: 'data-[state=open]:slide-in-from-left data-[state=closed]:slide-out-to-left inset-y-0 left-0 h-full w-3/4 border-r sm:max-w-sm',
                top: 'data-[state=open]:slide-in-from-top data-[state=closed]:slide-out-to-top inset-x-0 top-0 h-auto border-b',
                bottom: 'data-[state=open]:slide-in-from-bottom data-[state=closed]:slide-out-to-bottom inset-x-0 bottom-0 h-auto border-t',
            },
        },
        defaultVariants: {
            side: 'right',
        },
    },
)

type SheetContentProps = ComponentProps<typeof SheetContentPrimitive> &
    VariantProps<typeof sheetContentVariants> & {
        showCloseButton?: boolean
        showOverlay?: boolean
        /** Render through a portal to `document.body` (default). Set `false` to render inline — e.g. to dock the sheet inside a positioned ancestor rather than the viewport. */
        portal?: boolean
    }

export const SheetContent: FC<SheetContentProps> = ({
    className,
    children,
    side = 'right',
    showCloseButton = true,
    showOverlay = true,
    portal = true,
    ...props
}) => {
    const body = (
        <>
            {showOverlay && <SheetOverlay />}
            <SheetContentPrimitive
                data-slot="sheet-content"
                className={cn(sheetContentVariants({ side }), className)}
                {...props}
            >
                {children}
                {showCloseButton && (
                    <SheetClosePrimitive className="absolute top-4 right-4 rounded-md text-ink-subtle transition-colors hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary">
                        <X className="size-4" />
                        <span className="sr-only">Close</span>
                    </SheetClosePrimitive>
                )}
            </SheetContentPrimitive>
        </>
    )

    return portal ? <SheetPortal data-slot="sheet-portal">{body}</SheetPortal> : body
}

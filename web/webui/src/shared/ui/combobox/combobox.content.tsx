import type { ComponentProps, FC } from 'react'

import { Combobox as ComboboxPrimitive } from '@base-ui/react/combobox'

import { cn } from '@/shared/utils'

type ComboboxContentProps = ComponentProps<typeof ComboboxPrimitive.Popup> &
    Pick<
        ComponentProps<typeof ComboboxPrimitive.Positioner>,
        'side' | 'align' | 'sideOffset' | 'alignOffset' | 'anchor'
    >

const ComboboxContent: FC<ComboboxContentProps> = ({
    className,
    side = 'bottom',
    sideOffset = 6,
    align = 'start',
    alignOffset = 0,
    anchor,
    ...props
}) => (
    <ComboboxPrimitive.Portal>
        <ComboboxPrimitive.Positioner
            side={side}
            sideOffset={sideOffset}
            align={align}
            alignOffset={alignOffset}
            anchor={anchor}
            className="isolate z-50"
        >
            <ComboboxPrimitive.Popup
                data-slot="combobox-content"
                className={cn(
                    'group/combobox-content w-(--anchor-width) min-w-(--anchor-width) max-w-(--available-width) origin-(--transform-origin) overflow-hidden rounded-xl border border-hairline bg-canvas shadow-lg',
                    'fade-in-0 zoom-in-95 data-closed:fade-out-0 data-closed:zoom-out-95 animate-in data-closed:animate-out',
                    'data-[side=bottom]:slide-in-from-top-1 data-[side=top]:slide-in-from-bottom-1',
                    className,
                )}
                {...props}
            />
        </ComboboxPrimitive.Positioner>
    </ComboboxPrimitive.Portal>
)

export { ComboboxContent }

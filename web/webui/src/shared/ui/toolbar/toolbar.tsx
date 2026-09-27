import { type FC, type ReactNode, type Ref, useState } from 'react'

import { SlidersHorizontal } from 'lucide-react'

import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/shared/ui/sheet'
import { cn, useIsMobile } from '@/shared/utils'

type ToolbarProps = {
    children: ReactNode
    /** Controls kept out of the mobile Sheet: shown below the trigger on mobile, trailing the row on desktop. */
    actions?: ReactNode
    /** Mobile trigger label and Sheet title. */
    label?: string
    className?: string
    ref?: Ref<HTMLDivElement>
}

export const Toolbar: FC<ToolbarProps> = ({ children, actions, label = 'Filters', className, ref }) => {
    const isMobile = useIsMobile()
    const [open, setOpen] = useState(false)

    return (
        <Card ref={ref}>
            <CardContent className={cn('px-4 pt-2.5 pb-3', className)}>
                {isMobile ? (
                    <div className="flex flex-col gap-2 *:w-full">
                        <Sheet open={open} onOpenChange={setOpen}>
                            <SheetTrigger asChild>
                                <Button variant="outline" className="w-full">
                                    <SlidersHorizontal />
                                    {label}
                                </Button>
                            </SheetTrigger>
                            <SheetContent side="right">
                                <SheetHeader>
                                    <SheetTitle>{label}</SheetTitle>
                                </SheetHeader>
                                <div className="flex flex-col gap-3 p-4 *:w-full">{children}</div>
                            </SheetContent>
                        </Sheet>
                        {actions}
                    </div>
                ) : (
                    <div className="flex items-center gap-2">
                        {children}
                        {actions}
                    </div>
                )}
            </CardContent>
        </Card>
    )
}

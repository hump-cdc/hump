import type { FC, HTMLAttributes, Ref } from 'react'

import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

import { CardContext } from './card-context'

const cardVariants = cva('flex flex-col overflow-hidden rounded-xl border border-hairline bg-surface-1 text-ink', {
    variants: {
        separated: {
            true: 'gap-0 p-0',
            false: 'gap-3.5 p-4',
        },
    },
    defaultVariants: {
        separated: true,
    },
})

type CardProps = HTMLAttributes<HTMLDivElement> &
    VariantProps<typeof cardVariants> & {
        asChild?: boolean
        ref?: Ref<HTMLDivElement>
    }

export const Card: FC<CardProps> = ({ className, separated = true, asChild = false, ...props }) => {
    const Comp = asChild ? Slot : 'div'
    const isSeparated = separated ?? true

    return (
        <CardContext.Provider value={{ separated: isSeparated }}>
            <Comp className={cn(cardVariants({ separated: isSeparated, className }))} {...props} />
        </CardContext.Provider>
    )
}

import type { FC, HTMLAttributes, ReactNode, Ref } from 'react'

import { cva } from 'class-variance-authority'

import { cn } from '@/shared/utils'

import { useCardContext } from './card-context'

const cardFooterVariants = cva('flex shrink-0 items-center justify-between gap-2', {
    variants: {
        separated: {
            true: 'border-hairline border-t p-4',
            false: '',
        },
    },
    defaultVariants: {
        separated: true,
    },
})

type CardFooterProps = HTMLAttributes<HTMLDivElement> & {
    aside?: ReactNode
    ref?: Ref<HTMLDivElement>
}

export const CardFooter: FC<CardFooterProps> = ({ className, aside, children, ...props }) => {
    const { separated } = useCardContext()

    return (
        <div className={cn(cardFooterVariants({ separated, className }))} {...props}>
            {children}
            {aside ? <div className="flex items-center gap-2">{aside}</div> : null}
        </div>
    )
}

import type { FC, HTMLAttributes, ReactNode, Ref } from 'react'

import { cva } from 'class-variance-authority'

import { cn } from '@/shared/utils'

import { useCardContext } from './card-context'

const cardHeaderVariants = cva('flex shrink-0 items-center justify-between gap-2', {
    variants: {
        separated: {
            true: 'border-hairline border-b p-4',
            false: '',
        },
    },
    defaultVariants: {
        separated: true,
    },
})

type CardHeaderProps = HTMLAttributes<HTMLDivElement> & {
    aside?: ReactNode
    ref?: Ref<HTMLDivElement>
}

export const CardHeader: FC<CardHeaderProps> = ({ className, aside, children, ...props }) => {
    const { separated } = useCardContext()

    return (
        <div className={cn(cardHeaderVariants({ separated, className }))} {...props}>
            {children}
            {aside ? <div className="flex items-center gap-2">{aside}</div> : null}
        </div>
    )
}

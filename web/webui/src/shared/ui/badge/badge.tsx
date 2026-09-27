import type { FC, HTMLAttributes, ReactNode, Ref } from 'react'

import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

const badgeVariants = cva(
    'inline-flex min-h-6 select-none items-center gap-1.5 whitespace-nowrap border px-2 py-0.5 font-medium text-sm',
    {
        variants: {
            variant: {
                neutral: 'bg-surface-3 text-ink-muted [&_[data-slot=dot]]:bg-ink-subtle',
                healthy: 'bg-success-soft text-success',
                lag: 'bg-amber-soft text-amber',
                failed: 'bg-danger-soft text-danger',
                active: 'bg-primary-soft text-primary',
            },
            border: {
                true: 'rounded-full',
                false: 'rounded-md',
            },
        },
        compoundVariants: [
            { variant: 'neutral', border: false, class: 'border-surface-3' },
            { variant: 'healthy', border: false, class: 'border-success-soft' },
            { variant: 'lag', border: false, class: 'border-amber-soft' },
            { variant: 'failed', border: false, class: 'border-danger-soft' },
            { variant: 'active', border: false, class: 'border-primary-soft' },
            { variant: 'neutral', border: true, class: 'border-hairline' },
            { variant: 'healthy', border: true, class: 'border-success/30' },
            { variant: 'lag', border: true, class: 'border-amber/30' },
            { variant: 'failed', border: true, class: 'border-danger/30' },
            { variant: 'active', border: true, class: 'border-primary/30' },
        ],
        defaultVariants: {
            variant: 'neutral',
            border: false,
        },
    },
)

type BadgeProps = Omit<HTMLAttributes<HTMLSpanElement>, 'title'> &
    VariantProps<typeof badgeVariants> & {
        asChild?: boolean
        dot?: boolean
        dotAnimated?: boolean
        icon?: ReactNode
        title?: ReactNode
        ref?: Ref<HTMLSpanElement>
    }

export const Badge: FC<BadgeProps> = ({
    className,
    variant,
    border = false,
    asChild = false,
    dot = false,
    dotAnimated = false,
    icon,
    title,
    children,
    ...props
}) => {
    const Comp = asChild ? Slot : 'span'

    return (
        <Comp className={cn(badgeVariants({ variant, border, className }))} {...props}>
            {asChild ? (
                children
            ) : (
                <>
                    {dot && (
                        <span
                            data-slot="dot"
                            className={cn('size-1.5 shrink-0 rounded-full bg-current', dotAnimated && 'animate-ping')}
                        />
                    )}
                    {icon && (
                        <span data-slot="icon" className="inline-flex shrink-0 opacity-60 [&_svg]:size-3.5">
                            {icon}
                        </span>
                    )}
                    {title && (
                        <span data-slot="title" className="opacity-60">
                            {title}
                        </span>
                    )}
                    {children}
                </>
            )}
        </Comp>
    )
}

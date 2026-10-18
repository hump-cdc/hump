import type { FC, HTMLAttributes, ReactNode, Ref } from 'react'

import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

// A one-line minimum with centred content keeps the height the same whether it holds a number or the dash placeholder.
const infoCardContentVariants = cva('flex min-h-lh items-center break-all font-medium font-mono text-md', {
    variants: {
        variant: {
            neutral: 'text-ink',
            success: 'text-success',
            failed: 'text-danger',
            active: 'text-primary',
            lag: 'text-amber',
        },
    },
    defaultVariants: {
        variant: 'neutral',
    },
})

type InfoCardProps = Omit<HTMLAttributes<HTMLDivElement>, 'title'> &
    VariantProps<typeof infoCardContentVariants> & {
        /** Small heading above the content, e.g. the kind of the thing described; rendered uppercase. */
        title: string
        /** Secondary line under the content. */
        footer?: ReactNode
        ref?: Ref<HTMLDivElement>
    }

/** Bordered card naming one thing: a small title, monospace content tinted by `variant`, and an optional footer line. */
export const InfoCard: FC<InfoCardProps> = ({ title, footer, variant, className, children, ...props }) => (
    <div
        className={cn('flex flex-col gap-2.5 rounded-lg border border-hairline bg-surface-2/70 p-3.5', className)}
        {...props}
    >
        <span className="font-semibold text-2xs text-ink-tertiary uppercase tracking-widest">{title}</span>
        <div className="flex flex-col gap-0.5">
            <div className={infoCardContentVariants({ variant })}>{children}</div>
            {footer && <span className="text-ink-subtle text-xs">{footer}</span>}
        </div>
    </div>
)

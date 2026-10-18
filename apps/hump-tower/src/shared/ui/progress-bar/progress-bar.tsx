import type { FC, HTMLAttributes, Ref } from 'react'

import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

const trackVariants = cva('w-full overflow-hidden rounded-full bg-surface-3', {
    variants: {
        size: {
            sm: 'h-2',
            md: 'h-2.5',
        },
    },
    defaultVariants: {
        size: 'md',
    },
})

const fillVariants = cva('h-full rounded-full transition-[width] duration-500 ease-out motion-reduce:transition-none', {
    variants: {
        variant: {
            primary: 'bg-primary',
            success: 'bg-success',
            neutral: 'bg-ink-subtle',
        },
    },
    defaultVariants: {
        variant: 'primary',
    },
})

const clamp = (value: number): number => Math.min(100, Math.max(0, value))

type ProgressBarProps = Omit<HTMLAttributes<HTMLDivElement>, 'children'> &
    VariantProps<typeof trackVariants> &
    VariantProps<typeof fillVariants> & {
        /** Completion in the 0–100 range; values outside are clamped. */
        value: number
        ref?: Ref<HTMLDivElement>
    }

export const ProgressBar: FC<ProgressBarProps> = ({ className, value, variant, size, ...props }) => {
    const percent = clamp(value)

    return (
        <div
            role="progressbar"
            aria-valuenow={Math.round(percent)}
            aria-valuemin={0}
            aria-valuemax={100}
            className={cn(trackVariants({ size, className }))}
            {...props}
        >
            <div className={cn(fillVariants({ variant }))} style={{ width: `${percent}%` }} />
        </div>
    )
}

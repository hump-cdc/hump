import type { FC, HTMLAttributes } from 'react'

import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/utils'

const errorLineVariants = cva('', {
    variants: {
        tone: {
            default: 'text-[#f7f8f8]',
            muted: 'text-[#8a8f98]',
            danger: 'text-[#eb5757]',
            warning: 'text-[#f2c94c]',
        },
    },
    defaultVariants: {
        tone: 'default',
    },
})

type ErrorLineProps = HTMLAttributes<HTMLDivElement> & VariantProps<typeof errorLineVariants>

export const ErrorLine: FC<ErrorLineProps> = ({ className, tone, ...props }) => (
    <div className={cn(errorLineVariants({ tone, className }))} {...props} />
)

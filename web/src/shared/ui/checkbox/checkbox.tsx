import type { ComponentProps, FC, Ref } from 'react'

import { Indicator as CheckboxIndicator, Root as CheckboxRoot } from '@radix-ui/react-checkbox'
import { cva, type VariantProps } from 'class-variance-authority'
import { Check, Minus } from 'lucide-react'

import { cn } from '@/shared/utils'

const checkboxVariants = cva(
    'group peer inline-flex size-4.5 shrink-0 items-center justify-center rounded-[5px] border bg-transparent text-on-primary transition-colors outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-canvas disabled:pointer-events-none disabled:opacity-50 data-[state=checked]:border-primary data-[state=checked]:bg-primary data-[state=indeterminate]:border-hairline-strong data-[state=indeterminate]:bg-surface-3 data-[state=indeterminate]:text-ink-muted',
    {
        variants: {
            invalid: {
                false: 'border-hairline-strong',
                true: 'border-danger focus-visible:ring-danger aria-invalid:border-danger',
            },
        },
        defaultVariants: {
            invalid: false,
        },
    },
)

type CheckboxProps = ComponentProps<typeof CheckboxRoot> &
    VariantProps<typeof checkboxVariants> & {
        ref?: Ref<HTMLButtonElement>
    }

export const Checkbox: FC<CheckboxProps> = ({ className, invalid, ...props }) => (
    <CheckboxRoot
        aria-invalid={invalid || undefined}
        className={cn(checkboxVariants({ invalid, className }))}
        {...props}
    >
        <CheckboxIndicator className="inline-flex items-center justify-center text-current">
            <Check className="size-3.5 group-data-[state=indeterminate]:hidden" strokeWidth={3} />
            <Minus className="hidden size-3.5 group-data-[state=indeterminate]:block" strokeWidth={3} />
        </CheckboxIndicator>
    </CheckboxRoot>
)

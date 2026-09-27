import type { ButtonHTMLAttributes, FC, Ref } from 'react'

import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { LoaderCircle } from 'lucide-react'

import { cn } from '@/shared/utils'

const buttonVariants = cva(
    'inline-flex h-8.5 transform-gpu items-center justify-center gap-2 whitespace-nowrap rounded-lg px-3.5 font-medium text-base transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-canvas disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0',
    {
        variants: {
            variant: {
                primary: 'bg-primary text-on-primary hover:bg-primary-hover',
                secondary: 'bg-surface-3 text-ink-muted hover:bg-surface-2',
                outline: 'border border-hairline-strong bg-transparent text-ink-muted hover:bg-surface-1',
                ghost: 'bg-transparent text-ink-subtle hover:bg-surface-1 hover:text-ink-muted',
                danger: 'bg-danger text-white hover:opacity-90',
            },
            nonInteractive: {
                false: 'active:scale-[0.96] motion-reduce:active:scale-100',
                true: '',
            },
        },
        defaultVariants: {
            variant: 'primary',
            nonInteractive: false,
        },
    },
)

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> &
    VariantProps<typeof buttonVariants> & {
        asChild?: boolean
        loading?: boolean
        ref?: Ref<HTMLButtonElement>
    }

export const Button: FC<ButtonProps> = ({
    className,
    variant,
    nonInteractive,
    asChild = false,
    loading = false,
    disabled,
    children,
    ...props
}) => {
    const Comp = asChild ? Slot : 'button'

    return (
        <Comp
            className={cn(
                buttonVariants({ variant, nonInteractive, className }),
                loading && '[&_svg:not(.animate-spin)]:hidden',
            )}
            disabled={disabled || loading}
            {...props}
        >
            {asChild ? (
                children
            ) : (
                <>
                    {loading && <LoaderCircle className="animate-spin" />}
                    {children}
                </>
            )}
        </Comp>
    )
}

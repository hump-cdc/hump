import type { ComponentProps, CSSProperties, FC } from 'react'

import { cva } from 'class-variance-authority'
import { LoaderCircle } from 'lucide-react'
import { Toaster as ToasterPrimitive } from 'sonner'

export { toast } from 'sonner'

const toastVariants = cva('font-sans! [&_[data-icon]>svg]:size-4', {
    variants: {
        variant: {
            default: '[&_[data-icon]]:text-ink-subtle',
            success:
                '[--normal-bg:var(--success-soft)] [--normal-border:color-mix(in_oklab,var(--success)_30%,transparent)]',
            danger: '[--normal-bg:var(--danger-soft)] [--normal-border:color-mix(in_oklab,var(--danger)_30%,transparent)]',
            amber: '[--normal-bg:var(--amber-soft)] [--normal-border:color-mix(in_oklab,var(--amber)_30%,transparent)]',
        },
    },
    defaultVariants: {
        variant: 'default',
    },
})

const toasterStyle = {
    '--normal-bg': 'var(--surface-1)',
    '--normal-bg-hover': 'var(--surface-2)',
    '--normal-text': 'var(--ink)',
    '--normal-border': 'var(--hairline-strong)',
    '--normal-border-hover': 'var(--hairline-strong)',
    '--border-radius': '0.5rem',
} as CSSProperties

const toasterIcons = {
    success: null,
    info: null,
    warning: null,
    error: null,
    loading: <LoaderCircle className="animate-spin" />,
}

type ToasterProps = Omit<ComponentProps<typeof ToasterPrimitive>, 'closeButton'>

export const Toaster: FC<ToasterProps> = ({ position = 'bottom-center', style, toastOptions, ...props }) => (
    <ToasterPrimitive
        position={position}
        style={{ ...toasterStyle, ...style }}
        icons={toasterIcons}
        toastOptions={{
            ...toastOptions,
            classNames: {
                default: toastVariants({ variant: 'default' }),
                info: toastVariants({ variant: 'default' }),
                loading: toastVariants({ variant: 'default' }),
                success: toastVariants({ variant: 'success' }),
                error: toastVariants({ variant: 'danger' }),
                warning: toastVariants({ variant: 'amber' }),
                title: 'font-medium! text-ink!',
                description: 'text-ink-subtle!',
                actionButton: 'bg-primary! text-on-primary! hover:bg-primary-hover!',
                cancelButton: 'bg-surface-3! text-ink-muted!',
                ...toastOptions?.classNames,
            },
        }}
        {...props}
    />
)

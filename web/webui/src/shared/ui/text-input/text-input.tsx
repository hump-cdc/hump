import type { ChangeEvent, FC, ReactNode, Ref } from 'react'

import { X } from 'lucide-react'

import { cn } from '@/shared/utils'

import { useControllableValue, useMergedRef } from './text-input.helpers'

type TextInputProps = {
    value?: string
    defaultValue?: string
    onValueChange?: (value: string) => void
    placeholder?: string
    type?: 'text' | 'password'
    name?: string
    icon?: ReactNode
    invalid?: boolean
    disabled?: boolean
    font?: 'mono' | 'sans'
    className?: string
    ref?: Ref<HTMLInputElement>
}

export const TextInput: FC<TextInputProps> = ({
    value,
    defaultValue,
    onValueChange,
    placeholder,
    type = 'text',
    name,
    icon,
    invalid,
    disabled,
    font = 'mono',
    className,
    ref,
}) => {
    const { currentValue, hasValue, commit } = useControllableValue({ value, defaultValue, onValueChange })
    const { innerRef, setRefs } = useMergedRef(ref)

    const handleClear = () => {
        commit('')
        innerRef.current?.focus()
    }

    return (
        <div
            aria-invalid={invalid || undefined}
            className={cn(
                'flex h-9 w-full items-center gap-1.5 rounded-lg border border-hairline bg-canvas px-2.5 text-base outline-none transition-[color,box-shadow]',
                'focus-within:outline-none focus-within:ring-2 focus-within:ring-primary focus-within:ring-offset-2 focus-within:ring-offset-canvas',
                'aria-invalid:border-danger aria-invalid:focus-within:ring-danger',
                'has-[input:disabled]:pointer-events-none has-[input:disabled]:opacity-50',
                className,
            )}
        >
            {icon && <span className="flex shrink-0 items-center text-ink-subtle [&_svg]:size-4">{icon}</span>}

            <input
                ref={setRefs}
                value={currentValue}
                onChange={(event: ChangeEvent<HTMLInputElement>) => commit(event.target.value)}
                placeholder={placeholder}
                type={type}
                name={name}
                disabled={disabled}
                className={cn(
                    'h-full min-w-0 flex-1 bg-transparent text-ink outline-none placeholder:text-ink-tertiary',
                    font === 'mono' ? 'font-mono' : 'font-sans',
                )}
            />

            {hasValue && !disabled && (
                <button
                    type="button"
                    tabIndex={-1}
                    aria-label="Clear"
                    onClick={handleClear}
                    className="flex size-5 shrink-0 items-center justify-center rounded text-ink-subtle outline-none transition-colors hover:text-ink-muted focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                    <X className="size-3.5" />
                </button>
            )}
        </div>
    )
}

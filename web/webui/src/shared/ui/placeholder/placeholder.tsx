import type { FC, HTMLAttributes, ReactNode, Ref } from 'react'

import { cn } from '@/shared/utils'

type PlaceholderProps = Omit<HTMLAttributes<HTMLDivElement>, 'title'> & {
    icon?: ReactNode
    title: ReactNode
    description?: ReactNode
    ref?: Ref<HTMLDivElement>
}

/** Empty state: a round icon badge over a title and an optional description. */
export const Placeholder: FC<PlaceholderProps> = ({ className, icon, title, description, ...props }) => (
    <div
        className={cn(
            'flex min-h-0 flex-1 flex-col items-center justify-center gap-2 rounded-lg border border-hairline bg-canvas p-5 text-center',
            className,
        )}
        {...props}
    >
        {icon && (
            <span className="flex size-8.5 items-center justify-center rounded-full bg-surface-2 text-ink-tertiary [&_svg]:size-4">
                {icon}
            </span>
        )}
        <span className="font-medium text-base text-ink-subtle">{title}</span>
        {description && <span className="text-ink-tertiary text-xs">{description}</span>}
    </div>
)

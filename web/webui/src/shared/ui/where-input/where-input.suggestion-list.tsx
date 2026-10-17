import type { FC } from 'react'
import { useLayoutEffect, useRef } from 'react'

import { Popover } from '@base-ui/react/popover'
import { CaseSensitive, Columns3, Equal, SquareFunction } from 'lucide-react'

import { cn } from '@/shared/utils'

import type { WhereSuggestion, WhereSuggestionKind } from './where-input.completions'
import { MatchedLabel } from './where-input.matched-label'
import type { Suggestions } from './where-input.types'

const KIND_ICONS: Record<WhereSuggestionKind, FC<{ className?: string }>> = {
    field: Columns3,
    function: SquareFunction,
    keyword: CaseSensitive,
    operator: Equal,
}

type SuggestionListProps = {
    id: string
    open: boolean
    suggestions: Suggestions | null
    onActiveChange: (index: number) => void
    onAccept: (option: WhereSuggestion) => void
    onClose: () => void
}

export const SuggestionList: FC<SuggestionListProps> = ({
    id,
    open,
    suggestions,
    onActiveChange,
    onAccept,
    onClose,
}) => {
    const listRef = useRef<HTMLDivElement>(null)
    const active = suggestions?.active ?? 0

    useLayoutEffect(() => {
        listRef.current?.children[active]?.scrollIntoView({ block: 'nearest' })
    }, [active])

    return (
        <Popover.Root open={open} onOpenChange={(next) => !next && onClose()}>
            <Popover.Portal>
                <Popover.Positioner
                    anchor={suggestions?.anchor}
                    side="bottom"
                    align="start"
                    sideOffset={4}
                    className="isolate z-50"
                >
                    <Popover.Popup
                        initialFocus={false}
                        finalFocus={false}
                        // Keep focus in the editor so the panel stays expanded while picking with the mouse.
                        onMouseDown={(event) => event.preventDefault()}
                        className={cn(
                            'min-w-56 max-w-(--available-width) origin-(--transform-origin) overflow-hidden rounded-xl border border-hairline bg-canvas shadow-lg',
                            'fade-in-0 zoom-in-95 data-closed:fade-out-0 data-closed:zoom-out-95 animate-in data-closed:animate-out',
                            'data-[side=bottom]:slide-in-from-top-1 data-[side=top]:slide-in-from-bottom-1',
                        )}
                    >
                        <div ref={listRef} id={id} role="listbox" className="max-h-64 max-w-96 overflow-y-auto p-1">
                            {suggestions?.options.map((option, index) => {
                                const Icon = KIND_ICONS[option.kind]
                                return (
                                    // biome-ignore lint/a11y/useKeyWithClickEvents: keyboard selection is handled by the editor (aria-activedescendant)
                                    <div
                                        key={`${option.kind}:${option.label}`}
                                        id={`${id}-${index}`}
                                        role="option"
                                        tabIndex={-1}
                                        aria-selected={index === active}
                                        data-highlighted={index === active ? '' : undefined}
                                        onMouseMove={() => index !== active && onActiveChange(index)}
                                        onClick={() => onAccept(option)}
                                        className="flex cursor-default select-none items-center gap-2 rounded-md px-2 py-1.5 font-mono text-ink text-xs data-highlighted:bg-surface-1"
                                    >
                                        <Icon className="size-3.5 shrink-0 text-ink-subtle" />
                                        <MatchedLabel label={option.label} matches={option.matches} />
                                        {option.detail && (
                                            <span className="max-w-40 shrink-0 truncate text-ink-subtle">
                                                {option.detail}
                                            </span>
                                        )}
                                    </div>
                                )
                            })}
                        </div>
                    </Popover.Popup>
                </Popover.Positioner>
            </Popover.Portal>
        </Popover.Root>
    )
}

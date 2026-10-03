import type { FC, Ref } from 'react'
import { useId, useMemo } from 'react'

import { X } from 'lucide-react'
import { Editor } from 'prism-react-editor'

import { cn } from '@/shared/utils'

import { CaretTracker } from './where-input.caret-tracker'
import type { WhereField } from './where-input.completions'
import { NO_FIELDS } from './where-input.consts'
import { WHERE_LANGUAGE } from './where-input.grammar'
import { collapsedPreview } from './where-input.helpers'
import { useWhereInput } from './where-input.hook'
import { RestoreSuggestion } from './where-input.restore-suggestion'
import { SuggestionList } from './where-input.suggestion-list'

export type { WhereField } from './where-input.completions'

type WhereInputProps = {
    value: string
    onValueChange: (value: string) => void
    /** Called when the editor loses focus, e.g. to act on the finished condition. */
    onBlur?: () => void
    placeholder?: string
    /** Suggested condition offered for restore while the input is empty. */
    suggestion?: string
    /** Columns offered by autocompletion. */
    fields?: WhereField[]
    invalid?: boolean
    disabled?: boolean
    className?: string
    ref?: Ref<HTMLTextAreaElement>
}

/**
 * Single-line SQL condition input that expands in place into a multi-line editor while focused.
 * The expanded panel overlays surrounding content, so the parent layout never reflows.
 * Highlights PostgreSQL `WHERE` syntax and suggests columns, operators and functions while typing
 * (Ctrl+Space opens suggestions explicitly).
 */
export const WhereInput: FC<WhereInputProps> = ({
    value,
    onValueChange,
    onBlur,
    placeholder = '(no filter)',
    suggestion,
    fields = NO_FIELDS,
    invalid,
    disabled,
    className,
    ref,
}) => {
    const listId = useId()
    const {
        editorRef,
        initialValue,
        focused,
        expanded,
        height,
        suggestions,
        suggestionsOpen,
        setActive,
        accept,
        closeSuggestions,
        handleUpdate,
        handleSelectionChange,
        handleFocus,
        handleBlur,
        handlePanelMouseDown,
        handlePanelClick,
    } = useWhereInput({ value, onValueChange, onBlur, fields, ref })

    const hasValue = value.length > 0
    const preview = useMemo(() => collapsedPreview(value), [value])
    const activeOption = suggestionsOpen && suggestions ? `${listId}-${suggestions.active}` : undefined

    return (
        <div className={cn('relative h-9 w-full', className)}>
            {/* biome-ignore lint/a11y/noStaticElementInteractions lint/a11y/useKeyWithClickEvents: mouse handlers only redirect focus and keep clicks from reaching an enclosing label */}
            <div
                aria-invalid={invalid || undefined}
                onMouseDown={handlePanelMouseDown}
                onClick={handlePanelClick}
                style={{ height }}
                className={cn(
                    'absolute top-0 left-0 flex w-full min-w-0 items-start gap-1.5 overflow-hidden rounded-lg border border-hairline bg-canvas px-2.5 text-base',
                    'transition-[height,min-width,box-shadow] duration-200 ease-out motion-reduce:transition-none',
                    'focus-within:z-20 lg:focus-within:min-w-lg lg:focus-within:shadow-lg',
                    'focus-within:ring-2 focus-within:ring-primary focus-within:ring-offset-2 focus-within:ring-offset-canvas',
                    'aria-invalid:border-danger aria-invalid:focus-within:ring-danger',
                    'has-[textarea:disabled]:pointer-events-none has-[textarea:disabled]:opacity-50',
                )}
            >
                <span
                    className={cn(
                        'flex h-8.5 shrink-0 items-center font-mono text-xs',
                        invalid ? 'text-danger' : 'text-ink-subtle',
                    )}
                >
                    WHERE
                </span>

                <div className="relative h-full min-w-0 flex-1">
                    <Editor
                        ref={editorRef}
                        language={WHERE_LANGUAGE}
                        value={initialValue}
                        onUpdate={handleUpdate}
                        onSelectionChange={handleSelectionChange}
                        lineNumbers={false}
                        wordWrap={expanded}
                        readOnly={disabled}
                        className={cn(
                            'hump-where h-full font-mono text-ink',
                            !focused && preview && 'hump-where-collapsed',
                        )}
                        textareaProps={{
                            placeholder,
                            disabled,
                            onFocus: handleFocus,
                            onBlur: handleBlur,
                            role: 'combobox',
                            'aria-autocomplete': 'list',
                            'aria-expanded': suggestionsOpen,
                            'aria-controls': suggestionsOpen ? listId : undefined,
                            'aria-activedescendant': activeOption,
                        }}
                    >
                        <CaretTracker />
                    </Editor>

                    {/* Collapsed: one line, ellipsized; the editor's own lines are hidden (see `.hump-where-collapsed`). */}
                    {!focused && preview && (
                        <div className="hump-where pointer-events-none absolute inset-x-0 top-1.75 flex font-mono text-ink leading-5">
                            {/* The ellipsis takes the clipping box's color, so dim it there and restore the text color inside. */}
                            <span className="min-w-0 overflow-hidden text-ellipsis whitespace-pre text-ink-subtle">
                                <span
                                    className="text-ink"
                                    // biome-ignore lint/security/noDangerouslySetInnerHtml: Prism escapes the text while highlighting.
                                    dangerouslySetInnerHTML={{ __html: preview.html }}
                                />
                            </span>
                            {preview.more && <span className="shrink-0 pl-1 text-ink-subtle">…</span>}
                        </div>
                    )}
                </div>

                {focused && hasValue && !disabled && (
                    <span className="flex h-8.5 shrink-0 items-center">
                        <button
                            type="button"
                            tabIndex={-1}
                            aria-label="Clear"
                            onClick={() => onValueChange('')}
                            className="flex size-5 items-center justify-center rounded text-ink-subtle outline-none transition-colors hover:text-ink-muted focus-visible:ring-2 focus-visible:ring-primary/40"
                        >
                            <X className="size-3.5" />
                        </button>
                    </span>
                )}

                {focused && suggestion && !hasValue && !disabled && (
                    <span className="flex h-8.5 shrink-0 items-center">
                        <RestoreSuggestion suggestion={suggestion} onRestore={() => onValueChange(suggestion)} />
                    </span>
                )}
            </div>

            <SuggestionList
                id={listId}
                open={suggestionsOpen}
                suggestions={suggestions}
                onActiveChange={setActive}
                onAccept={accept}
                onClose={closeSuggestions}
            />
        </div>
    )
}

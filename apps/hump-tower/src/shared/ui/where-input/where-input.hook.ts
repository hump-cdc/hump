import type { MouseEvent, Ref } from 'react'
import { useLayoutEffect, useRef, useState } from 'react'

import type { PrismEditor } from 'prism-react-editor'
import { addHotkey, type Hotkey, type HotkeyMap, runHotkeys } from 'prism-react-editor/commands'
import { insertText } from 'prism-react-editor/utils'

import { useIsMobile } from '@/shared/utils'

import { getCompletions, type WhereField, type WhereSuggestion } from './where-input.completions'
import { COLLAPSED_HEIGHT } from './where-input.consts'
import { caretAnchor, clampHeight, contentHeightOf } from './where-input.helpers'
import type { Suggestions } from './where-input.types'

type UseWhereInputOptions = {
    value: string
    onValueChange: (value: string) => void
    onBlur?: () => void
    fields: WhereField[]
    ref?: Ref<HTMLTextAreaElement>
}

/**
 * Editor state and handlers for `WhereInput`: value sync, focus and panel height,
 * and the suggestion list with its keyboard navigation.
 */
export const useWhereInput = ({ value, onValueChange, onBlur, fields, ref }: UseWhereInputOptions) => {
    const editorRef = useRef<PrismEditor>(null)
    const [initialValue] = useState(value)
    const [focused, setFocused] = useState(false)
    // On mobile the input stays a single line even while focused.
    const isMobile = useIsMobile()
    const expanded = focused && !isMobile
    const [contentHeight, setContentHeight] = useState(0)
    const [suggestions, setSuggestions] = useState<Suggestions | null>(null)
    const [suggestionsOpen, setSuggestionsOpen] = useState(false)

    // Latest props and state for the editor's native key and update handlers.
    const latest = useRef({ value, onValueChange, fields, suggestions, suggestionsOpen })
    latest.current = { value, onValueChange, fields, suggestions, suggestionsOpen }
    /** Set by user input so the following selection change refreshes suggestions. */
    const typedRef = useRef(false)
    /** Suppresses suggestion refresh for programmatic edits (external value, inserted completion). */
    const silentRef = useRef(false)

    const closeSuggestions = () => {
        latest.current.suggestionsOpen = false
        setSuggestionsOpen(false)
    }

    const showSuggestions = (next: Suggestions) => {
        latest.current.suggestions = next
        latest.current.suggestionsOpen = true
        setSuggestions(next)
        setSuggestionsOpen(true)
    }

    const refreshSuggestions = (explicit: boolean) => {
        const editor = editorRef.current
        if (!editor) return
        const [start, end] = editor.getSelection()
        const completion = start === end ? getCompletions(editor.value, start, latest.current.fields, explicit) : null
        if (completion) {
            showSuggestions({ ...completion, active: 0, anchor: caretAnchor(editor) })
        } else {
            closeSuggestions()
        }
    }

    const setActive = (active: number) => {
        const current = latest.current.suggestions
        // A closing popup stays hoverable during its exit animation; don't reopen it.
        if (!current || !latest.current.suggestionsOpen) return
        const next = { ...current, active }
        latest.current.suggestions = next
        setSuggestions(next)
    }

    const accept = (option: WhereSuggestion) => {
        const editor = editorRef.current
        const current = latest.current.suggestions
        if (!editor || !current) return
        const caret = current.from + option.caret
        closeSuggestions()
        silentRef.current = true
        insertText(editor, option.insert, current.from, current.to, caret, caret)
        silentRef.current = false
    }

    const handleUpdate = (next: string) => {
        if (next !== latest.current.value) {
            latest.current.value = next
            latest.current.onValueChange(next)
        }
        if (!silentRef.current && editorRef.current?.focused) typedRef.current = true
    }

    const handleSelectionChange = () => {
        if (silentRef.current) return
        if (typedRef.current || latest.current.suggestionsOpen) refreshSuggestions(false)
        typedRef.current = false
    }

    // The editor owns its text after mount; apply only changes made outside it (clear, restore).
    useLayoutEffect(() => {
        const editor = editorRef.current
        if (!editor?.textarea || editor.value === value) return
        silentRef.current = true
        editor.textarea.value = value
        editor.update()
        silentRef.current = false
    }, [value])

    useLayoutEffect(() => {
        const textarea = editorRef.current?.textarea ?? null
        if (typeof ref === 'function') {
            ref(textarea)
        } else if (ref) {
            ref.current = textarea
        }
    }, [ref])

    // Hotkeys run in the capture phase, ahead of the editor's own key handling (e.g. Escape blurs),
    // so list navigation can swallow keys.
    // biome-ignore lint/correctness/useExhaustiveDependencies: handlers read the latest state through `latest`
    useLayoutEffect(() => {
        const container = editorRef.current?.container
        if (!container) return
        const hotkeys: HotkeyMap<null> = {}
        const whenOpen =
            (handler: (current: Suggestions) => void): Hotkey<null> =>
            () => {
                const current = latest.current.suggestions
                if (!latest.current.suggestionsOpen || !current) return false
                handler(current)
                return true
            }
        const move = (step: number) =>
            whenOpen((current) => setActive((current.active + step + current.options.length) % current.options.length))
        const acceptActive = whenOpen((current) => accept(current.options[current.active]))

        addHotkey(hotkeys, 'arrowdown', move(1))
        addHotkey(hotkeys, 'arrowup', move(-1))
        addHotkey(hotkeys, 'enter', acceptActive)
        addHotkey(hotkeys, 'tab', acceptActive)
        // When closed, Escape falls through to the editor, which blurs the textarea.
        addHotkey(hotkeys, 'escape', whenOpen(closeSuggestions))
        addHotkey(hotkeys, 'ctrl+ ', () => {
            refreshSuggestions(true)
            return true
        })

        const handleKeyDown = (event: KeyboardEvent) => runHotkeys(event, hotkeys, null)
        container.addEventListener('keydown', handleKeyDown, true)
        return () => container.removeEventListener('keydown', handleKeyDown, true)
    }, [])

    const measureContent = () => {
        const editor = editorRef.current
        if (editor) setContentHeight(contentHeightOf(editor))
    }

    // biome-ignore lint/correctness/useExhaustiveDependencies: re-measure when content or wrapping mode changes
    useLayoutEffect(measureContent, [value, focused])

    // Width animates on focus, so wrapping (and content height) changes while it runs.
    // biome-ignore lint/correctness/useExhaustiveDependencies: measureContent only reads refs
    useLayoutEffect(() => {
        const wrapper = editorRef.current?.wrapper
        if (!focused || !wrapper) return
        const observer = new ResizeObserver(measureContent)
        observer.observe(wrapper)
        return () => observer.disconnect()
    }, [focused])

    // Keep focus in the editor when clicking the panel chrome (label, buttons, padding).
    const handlePanelMouseDown = (event: MouseEvent<HTMLDivElement>) => {
        const textarea = editorRef.current?.textarea
        if (event.target === textarea) return
        event.preventDefault()
        textarea?.focus()
    }

    // Swallow the click so an enclosing `<label>` (e.g. a selectable row) is not activated.
    const handlePanelClick = (event: MouseEvent<HTMLDivElement>) => {
        event.preventDefault()
    }

    const handleBlur = () => {
        setFocused(false)
        closeSuggestions()
        const container = editorRef.current?.container
        if (container) {
            container.scrollTop = 0
            container.scrollLeft = 0
        }
        onBlur?.()
    }

    return {
        editorRef,
        initialValue,
        focused,
        expanded,
        height: expanded ? clampHeight(contentHeight) : COLLAPSED_HEIGHT,
        suggestions,
        suggestionsOpen,
        setActive,
        accept,
        closeSuggestions,
        handleUpdate,
        handleSelectionChange,
        handleFocus: () => setFocused(true),
        handleBlur,
        handlePanelMouseDown,
        handlePanelClick,
    }
}

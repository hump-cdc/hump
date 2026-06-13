import type { Ref } from 'react'
import { useRef, useState } from 'react'

type UseControllableValueOptions = {
    value?: string
    defaultValue?: string
    onValueChange?: (value: string) => void
}

type UseControllableValueResult = {
    currentValue: string
    hasValue: boolean
    commit: (next: string) => void
}

export const useControllableValue = ({
    value,
    defaultValue,
    onValueChange,
}: UseControllableValueOptions): UseControllableValueResult => {
    const [internalValue, setInternalValue] = useState(defaultValue ?? '')
    const currentValue = value ?? internalValue

    const commit = (next: string) => {
        if (value === undefined) {
            setInternalValue(next)
        }
        onValueChange?.(next)
    }

    return {
        currentValue,
        hasValue: currentValue.length > 0,
        commit,
    }
}

type UseMergedRefResult = {
    innerRef: React.RefObject<HTMLInputElement | null>
    setRefs: (node: HTMLInputElement | null) => void
}

export const useMergedRef = (ref?: Ref<HTMLInputElement>): UseMergedRefResult => {
    const innerRef = useRef<HTMLInputElement>(null)

    const setRefs = (node: HTMLInputElement | null) => {
        innerRef.current = node
        if (typeof ref === 'function') {
            ref(node)
        } else if (ref) {
            ref.current = node
        }
    }

    return { innerRef, setRefs }
}

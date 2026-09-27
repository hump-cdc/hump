import { useNavigate, useSearch } from '@tanstack/react-router'

import { useEffect } from 'react'

import { useOperationsFilter } from '@/entities/operations/filter'

import { operationsSearchSchema, toFilterValues } from './search'

/**
 * Presets the operations filter from the page's search params, resetting the facets they leave out
 * to "All", then drops them from the URL by replacing the history entry, so a reload or Back keeps
 * the (persisted) filter as the user left it.
 */
export const useSearchFilter = (): void => {
    const search = useSearch({ from: '/_sidebar/operations' })
    const navigate = useNavigate({ from: '/operations' })
    const apply = useOperationsFilter((state) => state.apply)

    useEffect(() => {
        const values = toFilterValues(search)
        if (values) {
            apply(values)
        }
        if (operationsSearchSchema.keyof().options.some((key) => key in search)) {
            void navigate({ search: {}, replace: true, viewTransition: false })
        }
    }, [search, apply, navigate])
}

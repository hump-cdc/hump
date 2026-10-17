import { z } from 'zod'

import {
    defaultFilterValues,
    type OperationsFilterValues,
    operationItems,
    statusItems,
} from '@/entities/operations/filter'

/**
 * One facet of the search: a single value or a repeated key. The default parser JSON-decodes
 * values, so a numeric-looking value arrives as a number; it is read back as a string. Values
 * are never comma-split, since targets can be URLs.
 */
const facetSchema = z
    .union([z.string(), z.number(), z.array(z.union([z.string(), z.number()]))])
    .transform((value): string[] => (Array.isArray(value) ? value : [value]).map(String))
    .optional()
    .catch(undefined)

/** Search params `/operations` accepts to preset its filter; each maps to one facet. */
export const operationsSearchSchema = z.object({
    table: facetSchema,
    sink: facetSchema,
    target: facetSchema,
    operation: facetSchema,
    result: facetSchema,
})

export type OperationsSearch = z.infer<typeof operationsSearchSchema>

/**
 * Filter the search presets, or `undefined` when it presets no facet. Facets the search leaves
 * out reset to "All" and the search text clears, so the page lists exactly what the params name.
 * Operation kinds and delivery results outside the known options are dropped, and a facet left
 * with none counts as left out.
 */
export const toFilterValues = (search: OperationsSearch): OperationsFilterValues | undefined => {
    const operations = pickKnown(search.operation, operationItems)
    const statuses = pickKnown(search.result, statusItems)
    if (!(search.table || search.sink || search.target || operations || statuses)) {
        return undefined
    }

    return {
        ...defaultFilterValues,
        ...(search.table && { tables: search.table }),
        ...(search.sink && { sinks: search.sink }),
        ...(search.target && { targets: search.target }),
        ...(operations && { operations }),
        ...(statuses && { statuses }),
    }
}

/** The `values` that name one of `items`, or `undefined` when none does. */
const pickKnown = (
    values: readonly string[] | undefined,
    items: readonly { value: string }[],
): string[] | undefined => {
    const known = (values ?? []).filter((value) => items.some((item) => item.value === value))
    return known.length > 0 ? known : undefined
}

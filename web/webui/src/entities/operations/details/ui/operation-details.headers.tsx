import type { ComponentProps, FC } from 'react'

import type { Header } from '../model/headers'
import { PropertyRow } from './operation-details.property-row'

/** Headers and the other metadata the sink reported, one copyable row each. */
export const Headers: FC<{ headers: Header[]; size?: ComponentProps<typeof PropertyRow>['size'] }> = ({
    headers,
    size,
}) => (
    <div className="flex flex-col">
        {headers.map((header) => (
            <PropertyRow
                key={header.key}
                size={size}
                label={header.key}
                value={header.value}
                copyValue={header.value}
                labelClassName="w-32.5 text-sm"
                valueClassName="text-ink-muted text-sm"
            />
        ))}
    </div>
)

import type { ComponentProps, FC } from 'react'

import { FileX } from 'lucide-react'

import { Placeholder } from './placeholder'

type NoPayloadPlaceholderProps = Omit<ComponentProps<typeof Placeholder>, 'icon' | 'title' | 'description'>

/** Empty state for an operation without a message body. */
export const NoPayloadPlaceholder: FC<NoPayloadPlaceholderProps> = (props) => (
    <Placeholder icon={<FileX />} title="No payload" description="No message body for the operation." {...props} />
)

import type { FC } from 'react'

/** Props every declarative node-config element shares. */
export type FlowNodeBase = {
    id: string
}

/**
 * A declarative node-config element (e.g. `<TableNode>`): a marker component that
 * renders nothing and carries a `flowNodeType` discriminator so the flow canvas can turn
 * it into a React Flow node bound to the matching registered renderer.
 */
export type FlowNodeConfig<T extends string, D> = FC<FlowNodeBase & D> & {
    flowNodeType: T
}

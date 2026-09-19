import { type FC, useEffect, useMemo } from 'react'

import { Search } from 'lucide-react'
import { useShallow } from 'zustand/react/shallow'

import { useConfig } from '@/entities/config/config/@x/operations'
import { MultiComboBox, type MultiComboBoxItem } from '@/shared/ui/multi-combo-box'
import { TextInput } from '@/shared/ui/text-input'

import { operationItems, statusItems } from '../model/options'
import { useOperationsFilter } from '../model/store'

export const OperationsFilter: FC = () => {
    const { data: config } = useConfig()

    const sinkItems = useMemo<MultiComboBoxItem[]>(
        () =>
            [...new Set((config?.sinks ?? []).map((sink) => sink.type))].map((type) => ({ value: type, label: type })),
        [config?.sinks],
    )
    const tableItems = useMemo<MultiComboBoxItem[]>(
        () =>
            (config?.tables ?? []).map((table) => {
                const name = `${table.schema}.${table.name}`
                return { value: name, label: name }
            }),
        [config?.tables],
    )

    const { sinks, tables, operations, statuses, search } = useOperationsFilter(
        useShallow((state) => ({
            sinks: state.sinks,
            tables: state.tables,
            operations: state.operations,
            statuses: state.statuses,
            search: state.search,
        })),
    )
    const { setSinks, setTables, setOperations, setStatuses, setSearch, hydrateSinks, hydrateTables } =
        useOperationsFilter(
            useShallow((state) => ({
                setSinks: state.setSinks,
                setTables: state.setTables,
                setOperations: state.setOperations,
                setStatuses: state.setStatuses,
                setSearch: state.setSearch,
                hydrateSinks: state.hydrateSinks,
                hydrateTables: state.hydrateTables,
            })),
        )

    useEffect(() => {
        hydrateSinks(sinkItems.map((item) => item.value))
    }, [sinkItems, hydrateSinks])
    useEffect(() => {
        hydrateTables(tableItems.map((item) => item.value))
    }, [tableItems, hydrateTables])

    return (
        <>
            <MultiComboBox
                items={sinkItems}
                value={sinks}
                onValueChange={setSinks}
                placeholder="Sinks"
                icon="Sinks"
                className="w-92"
            />
            <MultiComboBox
                items={tableItems}
                value={tables}
                onValueChange={setTables}
                placeholder="Tables"
                icon="Tables"
                className="w-92"
            />
            <MultiComboBox
                items={operationItems}
                value={operations}
                onValueChange={setOperations}
                placeholder="Operations"
                icon="Op"
                className="w-92"
            />
            <MultiComboBox
                items={statusItems}
                value={statuses}
                onValueChange={setStatuses}
                placeholder="Status"
                icon="Status"
                className="w-92"
            />

            <TextInput
                value={search}
                onValueChange={setSearch}
                placeholder="Search table, sink, error…"
                icon={<Search />}
                font="sans"
                className="w-full"
            />
        </>
    )
}

import { type FC, useMemo } from 'react'

import { BetweenHorizonalStart, Search, SlidersHorizontal, Table2, Waypoints, Workflow } from 'lucide-react'
import { useShallow } from 'zustand/react/shallow'

import { useConfig } from '@/entities/config/config/@x/operations'
import { useLiveControlsChartStep } from '@/entities/config/state/@x/operations'
import { useMetrics } from '@/entities/metrics/@x/operations'
import { MultiComboBox, type MultiComboBoxItem } from '@/shared/ui/multi-combo-box'
import { TextInput } from '@/shared/ui/text-input'

import { operationItems, statusItems } from '../model/options'
import { useOperationsFilter } from '../model/store'

export const OperationsFilter: FC = () => {
    const { data: config } = useConfig()
    const chartStep = useLiveControlsChartStep()
    const { data: metrics } = useMetrics(chartStep)

    const sinkItems = useMemo<MultiComboBoxItem[]>(
        () =>
            [...new Set((config?.sinks ?? []).map((sink) => sink.type))].map((type) => ({ value: type, label: type })),
        [config?.sinks],
    )
    const targetSinks = metrics?.topology.sinks
    const targetItems = useMemo<MultiComboBoxItem[]>(
        () =>
            [...new Set((targetSinks ?? []).flatMap((sink) => sink.targets.map((target) => target.target)))]
                .sort()
                .map((target) => ({ value: target, label: target })),
        [targetSinks],
    )
    const tableItems = useMemo<MultiComboBoxItem[]>(
        () =>
            (config?.tables ?? []).map((table) => {
                const name = `${table.schema}.${table.name}`
                return { value: name, label: name }
            }),
        [config?.tables],
    )

    const { sinks, targets, tables, operations, statuses, search } = useOperationsFilter(
        useShallow((state) => ({
            sinks: state.sinks,
            targets: state.targets,
            tables: state.tables,
            operations: state.operations,
            statuses: state.statuses,
            search: state.search,
        })),
    )
    const { setSinks, setTargets, setTables, setOperations, setStatuses, setSearch } = useOperationsFilter(
        useShallow((state) => ({
            setSinks: state.setSinks,
            setTargets: state.setTargets,
            setTables: state.setTables,
            setOperations: state.setOperations,
            setStatuses: state.setStatuses,
            setSearch: state.setSearch,
        })),
    )

    return (
        <div className="@container flex w-full flex-col gap-2">
            <div className="flex w-full @3xl:flex-row flex-col gap-2">
                <MultiComboBox
                    items={tableItems}
                    value={tables ?? allValues(tableItems)}
                    onValueChange={(value) => setTables(orAll(value, tableItems))}
                    placeholder="Tables"
                    hint="Tables whose changes are replicated"
                    icon={<Table2 />}
                    className="@3xl:w-1/3 w-full"
                />
                <MultiComboBox
                    items={sinkItems}
                    value={sinks ?? allValues(sinkItems)}
                    onValueChange={(value) => setSinks(orAll(value, sinkItems))}
                    placeholder="Sinks"
                    hint="Sink types the changes are delivered to"
                    icon={<Workflow />}
                    className="@3xl:w-1/3 w-full"
                />
                <MultiComboBox
                    items={targetItems}
                    value={targets ?? allValues(targetItems)}
                    onValueChange={(value) => setTargets(orAll(value, targetItems))}
                    placeholder="Targets"
                    hint="Topics, subjects, exchanges, streams or URLs the sinks deliver to"
                    icon={<BetweenHorizonalStart />}
                    className="@3xl:w-1/3 w-full"
                />
            </div>

            <div className="flex w-full @3xl:flex-row flex-col gap-2">
                <MultiComboBox
                    items={operationItems}
                    value={operations}
                    onValueChange={setOperations}
                    placeholder="Operations"
                    hint="Row operations: insert, update, delete or truncate"
                    icon={<Waypoints />}
                    className="@3xl:w-92 w-full"
                />
                <MultiComboBox
                    items={statusItems}
                    value={statuses}
                    onValueChange={setStatuses}
                    placeholder="Status"
                    hint="Delivery result: delivered or failed"
                    icon={<SlidersHorizontal />}
                    className="@3xl:w-92 w-full"
                />
                <TextInput
                    value={search}
                    onValueChange={setSearch}
                    placeholder="Search payload, target, error…"
                    icon={<Search />}
                    font="sans"
                    className="w-full"
                />
            </div>
        </div>
    )
}

/** Values of every option. */
const allValues = (items: readonly MultiComboBoxItem[]): string[] => items.map((item) => item.value)

/** `selected`, or `null` when it includes every option so options that load later stay allowed. */
const orAll = (selected: string[], items: readonly MultiComboBoxItem[]): string[] | null =>
    items.every((item) => selected.includes(item.value)) ? null : selected

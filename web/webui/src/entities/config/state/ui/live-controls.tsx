import type { FC } from 'react'

import { useShallow } from 'zustand/react/shallow'

import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from '@/shared/ui/dropdown-menu'
import { Tabs, TabsList, TabsTrigger } from '@/shared/ui/tabs'

import { CHART_STEPS, type ChartStep, useLiveControlsStore } from '../model/store'
import { LiveStatus } from './live-status'
import { SnapshotStatus } from './snapshot-status'

export const LiveControls: FC = () => {
    const { chartStep, setChartStep } = useLiveControlsStore(
        useShallow((state) => ({
            chartStep: state.chartStep,
            setChartStep: state.setChartStep,
        })),
    )

    return (
        <>
            <SnapshotStatus />
            <DropdownMenu>
                <DropdownMenuTrigger asChild>
                    <LiveStatus className="cursor-pointer" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="flex w-56 flex-col gap-3 p-3">
                    <div className="flex flex-col gap-2">
                        <span className="font-semibold text-ink-subtle text-xs uppercase tracking-wide">
                            Chart step
                        </span>
                        <Tabs value={chartStep} onValueChange={(value) => setChartStep(value as ChartStep)}>
                            <TabsList className="w-full">
                                {CHART_STEPS.map((option) => (
                                    <TabsTrigger key={option} value={option} className="flex-1 py-1 text-xs">
                                        {option}
                                    </TabsTrigger>
                                ))}
                            </TabsList>
                        </Tabs>
                    </div>
                </DropdownMenuContent>
            </DropdownMenu>
        </>
    )
}
